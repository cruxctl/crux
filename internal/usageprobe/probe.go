package usageprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/cruxctl/crux/internal/envpath"
	"github.com/cruxctl/crux/pkg/cruxapi"
)

const maxMetricValueBytes = 2000

var (
	emailPattern = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	uuidPattern  = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
)

func Probe(ctx context.Context, agent cruxapi.Agent, timeout time.Duration) []cruxapi.UsageMetric {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	name := cruxapi.CleanAgentName(agent.Name)
	metrics := []cruxapi.UsageMetric{
		CommandMetric(ctx, agent, timeout, "cli.version", "Installed CLI version.", "--version"),
	}
	switch name {
	case "claude":
		metrics = append(metrics,
			CommandMetric(ctx, agent, timeout, "auth", "Claude authentication status from `claude auth status`.", "auth", "status"),
			CommandMetric(ctx, agent, timeout, "subscription", "Claude Code subscription and reset information from `claude -p /usage`.", "-p", "/usage"),
			cruxapi.UsageMetric{Name: "tokens", Available: false, Description: "Claude Code does not expose normalized token counters through a stable non-interactive command; captured run output remains the Crux-owned source."},
		)
	case "codex":
		metrics = append(metrics,
			CommandMetric(ctx, agent, timeout, "auth", "Codex login state from `codex login status`.", "login", "status"),
			cruxapi.UsageMetric{Name: "tokens", Available: false, Description: "Codex does not expose aggregate token counters through a stable status command; per-session token text is only available when printed by Codex."},
			cruxapi.UsageMetric{Name: "sandbox", Available: true, Value: "read from command invocation and Codex config, not normalized in Crux yet", Description: "Sandbox/approval policy is execution configuration rather than account usage."},
		)
	case "gemini":
		metrics = append(metrics,
			CommandMetric(ctx, agent, timeout, "sessions", "Gemini local session list from `gemini --list-sessions`.", "--list-sessions"),
			cruxapi.UsageMetric{Name: "tokens", Available: false, Description: "Gemini CLI does not expose aggregate account token/quota counters through a stable local command."},
		)
	case "kimi":
		metrics = append(metrics,
			CommandMetric(ctx, agent, timeout, "info", "Kimi CLI and protocol information from `kimi info`.", "info"),
			cruxapi.UsageMetric{Name: "membership", Available: false, Description: "Kimi membership/quota state is provider-side unless surfaced in command output."},
		)
	default:
		metrics = append(metrics, cruxapi.UsageMetric{Name: "external", Available: false, Description: "No live usage probe is registered for this custom agent."})
	}
	return metrics
}

func CommandMetric(ctx context.Context, agent cruxapi.Agent, timeout time.Duration, name, description string, args ...string) cruxapi.UsageMetric {
	out, err := RunCommand(ctx, agent, timeout, args...)
	if err != nil {
		return cruxapi.UsageMetric{Name: name, Available: false, Value: err.Error(), Description: description}
	}
	return cruxapi.UsageMetric{Name: name, Available: true, Value: out, Description: description}
}

func RunCommand(parent context.Context, agent cruxapi.Agent, timeout time.Duration, args ...string) (string, error) {
	return RunCommandWithSanitizer(parent, agent, timeout, Sanitize, args...)
}

func RunCommandWithSanitizer(parent context.Context, agent cruxapi.Agent, timeout time.Duration, sanitizer func(string) string, args ...string) (string, error) {
	if strings.TrimSpace(agent.Command.Path) == "" {
		return "", fmt.Errorf("agent command path is empty")
	}
	if sanitizer == nil {
		sanitizer = Sanitize
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, agent.Command.Path, args...)
	cmd.Env = envpath.CommandEnv(os.Environ(), agent.Command.Path, agent.Command.Env)
	if strings.TrimSpace(agent.Command.WorkingDir) != "" {
		cmd.Dir = agent.Command.WorkingDir
	}
	data, err := cmd.CombinedOutput()
	value := sanitizer(string(data))
	if ctx.Err() == context.DeadlineExceeded {
		return value, fmt.Errorf("probe timed out after %s", timeout)
	}
	if err != nil {
		if value != "" {
			return value, fmt.Errorf("%w: %s", err, value)
		}
		return value, err
	}
	return value, nil
}

func Sanitize(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = redactStructuredValue(value)
	value = emailPattern.ReplaceAllString(value, "<redacted-email>")
	value = uuidPattern.ReplaceAllString(value, "<redacted-id>")
	if len(value) <= maxMetricValueBytes {
		return value
	}
	return value[:maxMetricValueBytes] + "...(truncated)"
}

func redactStructuredValue(value string) string {
	if value == "" || !json.Valid([]byte(value)) {
		return value
	}
	var data any
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return value
	}
	redactJSON(data, "")
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return value
	}
	return strings.TrimSpace(out.String())
}

func redactJSON(value any, key string) {
	switch typed := value.(type) {
	case map[string]any:
		for childKey, childValue := range typed {
			if isSensitiveKey(childKey) {
				typed[childKey] = "<redacted>"
				continue
			}
			redactJSON(childValue, childKey)
		}
	case []any:
		for _, item := range typed {
			redactJSON(item, key)
		}
	}
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
	for _, part := range []string{
		"accountid",
		"apikey",
		"email",
		"orgid",
		"orgname",
		"password",
		"refreshtoken",
		"secret",
		"session",
		"token",
		"userid",
		"username",
	} {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}
