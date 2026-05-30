package managedops

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

const (
	HistorySourceCrux     = "crux"
	HistorySourceProvider = "provider"
)

func Capabilities(agent cruxapi.Agent) cruxapi.AgentCapabilities {
	name := cruxapi.CleanAgentName(agent.Name)
	capability := cruxapi.AgentCapabilities{
		AgentName:   agent.Name,
		Provider:    providerName(name),
		UsageProbe:  false,
		CruxHistory: true,
		Fallback:    true,
	}
	switch name {
	case "claude":
		capability.CostSignals = true
		capability.ProviderSessions = true
		capability.Resume = true
		capability.TTYExec = true
		capability.SessionListCommand = []string{"tty", "transcript-backed exec records"}
		capability.ResumeCommandTemplate = []string{"claude", "--resume", "<session>"}
		capability.TTYExecCommand = []string{"claude"}
		capability.Notes = append(capability.Notes,
			"Claude usage, cost, and session evidence is captured through TTY transcripts imported by Crux.",
			"Claude sessions are resumed through the interactive TTY path with `claude --resume <session>` or `claude --continue`.",
		)
	case "codex":
		capability.ProviderSessions = true
		capability.Resume = true
		capability.TTYExec = true
		capability.SessionListCommand = []string{"tty", "transcript-backed exec records"}
		capability.ResumeCommandTemplate = []string{"codex", "resume", "--all", "<session>"}
		capability.TTYExecCommand = []string{"codex"}
		capability.Notes = append(capability.Notes,
			"Codex usage, cost, and session evidence is captured through TTY transcripts imported by Crux.",
			"Codex sessions are resumed through the interactive TTY path with `codex resume --all <session>`.",
		)
	case "gemini":
		capability.ProviderSessions = true
		capability.Resume = true
		capability.TTYExec = true
		capability.SessionListCommand = []string{"tty", "transcript-backed exec records"}
		capability.ResumeCommandTemplate = []string{"gemini", "--skip-trust", "--resume", "<session>"}
		capability.TTYExecCommand = []string{"gemini", "--skip-trust"}
		capability.Notes = append(capability.Notes, "Gemini usage, cost, and session evidence is captured through TTY transcripts imported by Crux.")
	case "kimi":
		capability.CostSignals = true
		capability.ProviderSessions = true
		capability.Resume = true
		capability.TTYExec = true
		capability.SessionListCommand = []string{"tty", "transcript-backed exec records"}
		capability.ResumeCommandTemplate = []string{"kimi", "--session", "<session>"}
		capability.TTYExecCommand = []string{"kimi"}
		capability.Notes = append(capability.Notes, "Kimi usage, cost, and session evidence is captured through TTY transcripts imported by Crux.")
	default:
		capability.UsageProbe = false
		capability.TTYExec = true
		capability.Notes = append(capability.Notes, "Custom command-backed agents rely on Crux-owned execution history unless they add a future capability adapter.")
	}
	return capability
}

func ExecAgent(agent cruxapi.Agent, req cruxapi.AgentExecPlanRequest) (cruxapi.Agent, error) {
	planned := agent
	session := strings.TrimSpace(req.ResumeSession)
	prompt := strings.TrimSpace(req.Prompt)
	name := cruxapi.CleanAgentName(agent.Name)
	switch name {
	case "claude":
		planned.Command.Args = resumeArgs(session, []string{"--continue"}, []string{"--resume", session})
	case "codex":
		planned.Command.Args = resumeArgs(session, []string{"resume", "--all", "--last"}, []string{"resume", "--all", session})
	case "gemini":
		if session == "last" {
			session = "latest"
		}
		planned.Command.Args = []string{"--skip-trust"}
		if session != "" && session != "new" {
			planned.Command.Args = append(planned.Command.Args, "--resume", session)
		}
		planned.Command.Env = mergeEnv(planned.Command.Env, map[string]string{
			"COLORTERM": "truecolor",
			"TERM":      "xterm-256color",
		})
	case "kimi":
		planned.Command.Args = resumeArgs(session, []string{"--continue"}, []string{"--session", session})
	default:
		planned.Command.Args = append([]string{}, planned.Command.Args...)
		if prompt != "" {
			planned.Command = substitutePrompt(planned.Command, prompt)
		}
	}
	planned.Command.Args = append(planned.Command.Args, req.Args...)
	if workingDir := strings.TrimSpace(req.WorkingDir); workingDir != "" {
		planned.Command.WorkingDir = workingDir
	}
	return planned, nil
}

func ExecInput(agent cruxapi.Agent, req cruxapi.AgentExecPlanRequest, planned cruxapi.Agent) []string {
	input := append([]string{}, req.Input...)
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return input
	}
	if cruxapi.CleanAgentName(agent.Name) == "custom" || !isManagedProvider(agent.Name) {
		if commandHasPromptPlaceholder(agent.Command) && !commandHasPromptPlaceholder(planned.Command) {
			return input
		}
	}
	return append(input, prompt)
}

func ResumeAgent(agent cruxapi.Agent, sessionID string) (cruxapi.Agent, error) {
	session := strings.TrimSpace(sessionID)
	if session == "" {
		return agent, fmt.Errorf("resume session is required")
	}
	name := cruxapi.CleanAgentName(agent.Name)
	resumed := agent
	switch name {
	case "claude":
		if session == "last" || session == "latest" {
			resumed.Command.Args = []string{"--continue", "-p", "{prompt}"}
		} else {
			resumed.Command.Args = []string{"--resume", session, "-p", "{prompt}"}
		}
	case "codex":
		if session == "last" || session == "latest" {
			resumed.Command.Args = []string{"exec", "resume", "--all", "--last", "--skip-git-repo-check", "{prompt}"}
		} else {
			resumed.Command.Args = []string{"exec", "resume", "--all", session, "--skip-git-repo-check", "{prompt}"}
		}
	case "gemini":
		if session == "last" {
			session = "latest"
		}
		resumed.Command.Args = []string{"--skip-trust", "--resume", session, "-p", "{prompt}"}
		resumed.Command.Env = mergeEnv(resumed.Command.Env, map[string]string{
			"COLORTERM": "truecolor",
			"TERM":      "xterm-256color",
		})
	case "kimi":
		if session == "last" || session == "latest" {
			resumed.Command.Args = []string{"--quiet", "--continue", "--prompt", "{prompt}"}
		} else {
			resumed.Command.Args = []string{"--quiet", "--session", session, "--prompt", "{prompt}"}
		}
	default:
		return agent, fmt.Errorf("agent %q does not have a resume adapter", agent.Name)
	}
	return resumed, nil
}

func resumeArgs(session string, latestArgs, sessionArgs []string) []string {
	switch session {
	case "", "new":
		return nil
	case "last", "latest":
		return append([]string{}, latestArgs...)
	default:
		return append([]string{}, sessionArgs...)
	}
}

func isManagedProvider(name string) bool {
	switch cruxapi.CleanAgentName(name) {
	case "claude", "codex", "gemini", "kimi":
		return true
	default:
		return false
	}
}

func substitutePrompt(command cruxapi.CommandSpec, prompt string) cruxapi.CommandSpec {
	out := command
	out.Args = append([]string{}, command.Args...)
	for i, arg := range out.Args {
		out.Args[i] = strings.ReplaceAll(arg, "{prompt}", prompt)
	}
	if len(command.Env) > 0 {
		out.Env = make(map[string]string, len(command.Env))
		for key, value := range command.Env {
			out.Env[key] = strings.ReplaceAll(value, "{prompt}", prompt)
		}
	}
	return out
}

func commandHasPromptPlaceholder(command cruxapi.CommandSpec) bool {
	for _, arg := range command.Args {
		if strings.Contains(arg, "{prompt}") {
			return true
		}
	}
	for _, value := range command.Env {
		if strings.Contains(value, "{prompt}") {
			return true
		}
	}
	return false
}

func Sessions(ctx context.Context, agent cruxapi.Agent, executions []cruxapi.Execution, timeout time.Duration) []cruxapi.AgentSession {
	capability := Capabilities(agent)
	sessions := make([]cruxapi.AgentSession, 0)
	for _, execution := range executions {
		if execution.AgentName != agent.Name {
			continue
		}
		title := preview(execution.Prompt, 80)
		if title == "" {
			title = execution.ID
		}
		sessions = append(sessions, cruxapi.AgentSession{
			AgentName:       agent.Name,
			Provider:        capability.Provider,
			ID:              execution.ID,
			Title:           title,
			WorkingDir:      execution.WorkingDir,
			Source:          HistorySourceCrux,
			ResumeSupported: false,
			UpdatedAt:       executionUpdatedAt(execution),
		})
	}
	if len(sessions) == 0 && !capability.ProviderSessions {
		sessions = append(sessions, cruxapi.AgentSession{
			AgentName: agent.Name,
			Provider:  capability.Provider,
			ID:        "tty-session-list-empty",
			Title:     "No transcript-backed TTY sessions have been recorded for this agent.",
			Source:    HistorySourceProvider,
			Raw:       strings.Join(capability.Notes, " "),
		})
	}
	return sessions
}

func CostSnapshot(usage cruxapi.AgentUsage) cruxapi.AgentCostSnapshot {
	snapshot := cruxapi.AgentCostSnapshot{
		AgentName:              usage.AgentName,
		Provider:               providerName(cruxapi.CleanAgentName(usage.AgentName)),
		Status:                 usage.Status,
		ExecutionsTotal:        usage.ExecutionsTotal,
		Queued:                 usage.Queued,
		Running:                usage.Running,
		Succeeded:              usage.Succeeded,
		Failed:                 usage.Failed,
		SuccessRate:            usage.SuccessRate,
		StdoutBytes:            usage.StdoutBytes,
		StderrBytes:            usage.StderrBytes,
		TotalDurationSeconds:   usage.TotalDurationSeconds,
		AverageDurationSeconds: usage.AverageDurationSeconds,
		LastExecutionID:        usage.LastExecutionID,
		LastStatus:             usage.LastStatus,
		LastError:              usage.LastError,
		ExternalMetrics:        usage.ExternalMetrics,
		Notes:                  []string{"Realtime Crux cost monitoring is transcript-backed: running/queued work, duration, output bytes, exit status, and provider usage text captured through TTY sessions."},
	}
	for _, metric := range usage.ExternalMetrics {
		if isCostMetric(metric) {
			snapshot.ProviderCostAvailable = metric.Available
			snapshot.ProviderCostValue = metric.Value
			snapshot.ProviderCostDescription = metric.Description
			break
		}
	}
	if !snapshot.ProviderCostAvailable && snapshot.ProviderCostDescription == "" {
		snapshot.ProviderCostDescription = "Provider-side cost or quota counters are only available when captured in imported TTY transcripts for this agent."
	}
	return snapshot
}

func History(executions []cruxapi.Execution, agentName string) []cruxapi.AgentHistoryItem {
	items := make([]cruxapi.AgentHistoryItem, 0)
	for _, execution := range executions {
		if execution.AgentName != agentName {
			continue
		}
		items = append(items, cruxapi.AgentHistoryItem{
			ID:             execution.ID,
			AgentName:      execution.AgentName,
			Status:         execution.Status,
			ExitCode:       execution.ExitCode,
			Prompt:         execution.Prompt,
			PromptPreview:  preview(execution.Prompt, 96),
			StdoutPreview:  preview(execution.Stdout, 96),
			StderrPreview:  preview(execution.Stderr, 96),
			Error:          execution.Error,
			WorkingDir:     execution.WorkingDir,
			ResumeSession:  execution.ResumeSession,
			SourceExecID:   execution.SourceExecID,
			FallbackAgents: execution.FallbackAgents,
			QueuedAt:       execution.QueuedAt,
			StartedAt:      execution.StartedAt,
			CompletedAt:    execution.CompletedAt,
		})
	}
	return items
}

func executionUpdatedAt(execution cruxapi.Execution) *time.Time {
	if execution.CompletedAt != nil {
		return execution.CompletedAt
	}
	if execution.StartedAt != nil {
		return execution.StartedAt
	}
	t := execution.QueuedAt
	return &t
}

func isCostMetric(metric cruxapi.UsageMetric) bool {
	switch metric.Name {
	case "subscription", "tokens", "membership":
		return true
	default:
		return false
	}
}

func providerName(name string) string {
	switch name {
	case "claude", "codex", "gemini", "kimi":
		return name
	default:
		return "custom"
	}
}

func mergeEnv(base, extra map[string]string) map[string]string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}

func preview(value string, limit int) string {
	value = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\n", " | "))
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "...(truncated)"
}

func sanitizeSessionList(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\r\n", "\n")
	if len(value) <= 4000 {
		return value
	}
	return value[:4000] + "...(truncated)"
}
