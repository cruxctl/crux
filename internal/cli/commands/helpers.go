package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cruxctl/crux/internal/agent"
	"github.com/cruxctl/crux/internal/client"
	"github.com/cruxctl/crux/internal/config"
	"github.com/cruxctl/cruxd/pkg/cruxapi"
	"gopkg.in/yaml.v3"
)

func buildClient(opts Options) (*client.Client, config.CLIContext, error) {
	cfg, _, err := config.LoadCLIConfig(opts.ConfigPath)
	if err != nil {
		return nil, config.CLIContext{}, err
	}
	ctxCfg, _, err := cfg.ActiveContext(opts.ContextName)
	if err != nil {
		return nil, config.CLIContext{}, err
	}
	if env := strings.TrimSpace(os.Getenv("CRUX_SERVER_URL")); env != "" {
		ctxCfg.ServerURL = env
	}
	if env := strings.TrimSpace(os.Getenv("CRUX_API_KEY")); env != "" {
		ctxCfg.APIKey = env
	}
	if opts.ServerURL != "" {
		ctxCfg.ServerURL = opts.ServerURL
	}
	if opts.APIKey != "" {
		ctxCfg.APIKey = opts.APIKey
	}
	return client.New(ctxCfg.ServerURL, ctxCfg.APIKey), ctxCfg, nil
}

func printOutput(opts Options, value any) error {
	switch opts.Format {
	case "", "table", "json":
		encoder := json.NewEncoder(opts.Out)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	case "yaml":
		data, err := yaml.Marshal(value)
		if err != nil {
			return err
		}
		fmt.Fprint(opts.Out, string(data))
	default:
		return fmt.Errorf("unsupported output format %q", opts.Format)
	}
	return nil
}

func printEventTable(out io.Writer, events []cruxapi.Event, includeTarget bool) {
	if includeTarget {
		fmt.Fprintf(out, "%-20s %-29s %-18s %-24s %-8s %s\n", "TIME", "EXECUTION", "AGENT", "TYPE", "EXIT", "MESSAGE")
	} else {
		fmt.Fprintf(out, "%-20s %-18s %-24s %-8s %s\n", "TIME", "AGENT", "TYPE", "EXIT", "MESSAGE")
	}
	for _, event := range events {
		exitCode := eventDataString(event, "exitCode")
		if includeTarget {
			fmt.Fprintf(out, "%-20s %-29s %-18s %-24s %-8s %s\n",
				event.CreatedAt.Format(time.RFC3339), firstNonEmpty(event.ExecutionID, "-"), firstNonEmpty(event.AgentName, "-"), event.Type, firstNonEmpty(exitCode, "-"), event.Message)
			continue
		}
		fmt.Fprintf(out, "%-20s %-18s %-24s %-8s %s\n",
			event.CreatedAt.Format(time.RFC3339), firstNonEmpty(event.AgentName, "-"), event.Type, firstNonEmpty(exitCode, "-"), event.Message)
	}
}

func eventDataString(event cruxapi.Event, key string) string {
	if event.Data == nil {
		return ""
	}
	value, ok := event.Data[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case int:
		return strconv.Itoa(typed)
	case float64:
		return strconv.Itoa(int(typed))
	default:
		return fmt.Sprint(typed)
	}
}

func firstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func printRuntimeConfigTable(out io.Writer, runtime cruxapi.RuntimeConfig) {
	fmt.Fprintf(out, "%-28s %s\n", "KEY", "VALUE")
	fmt.Fprintf(out, "%-28s %d\n", "workerConcurrency", runtime.WorkerConcurrency)
	fmt.Fprintf(out, "%-28s %d\n", "jobTimeoutSeconds", runtime.JobTimeoutSeconds)
	fmt.Fprintf(out, "%-28s %d\n", "maxOutputBytes", runtime.MaxOutputBytes)
	fmt.Fprintf(out, "%-28s %d\n", "discoveryTimeoutSeconds", runtime.DiscoveryTimeoutSecs)
	fmt.Fprintf(out, "%-28s %s\n", "logLevel", runtime.LogLevel)
	fmt.Fprintf(out, "%-28s %s\n", "defaultNamespace", runtime.DefaultNamespace)
	fmt.Fprintf(out, "%-28s %t\n", "allowShellCommands", runtime.AllowShellCommands)
	fmt.Fprintf(out, "%-28s %d\n", "traceRetentionEntries", runtime.TraceRetentionEntries)
}

func printAgentStateTable(out io.Writer, states []agent.AgentState) {
	sort.Slice(states, func(i, j int) bool { return states[i].ID < states[j].ID })
	fmt.Fprintf(out, "%-18s %-10s %-48s %s\n", "NAME", "STATUS", "COMMAND", "VERSION")
	for _, state := range states {
		command := "-"
		if state.BinaryPath != "" {
			command = state.BinaryPath
		}
		fmt.Fprintf(out, "%-18s %-10s %-48s %s\n", state.ID, state.Status, compactPathWidth(command, 48), oneLine(state.Version))
	}
}

func oneLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " | ")
	if len(value) > 240 {
		return value[:240] + "...(truncated)"
	}
	return value
}

func compactPathWidth(value string, width int) string {
	value = oneLine(value)
	if len(value) <= width {
		return value
	}
	if width <= 3 {
		return value[:width]
	}
	return "..." + value[len(value)-(width-3):]
}
