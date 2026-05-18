package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cruxctl/crux/internal/agent"
	"github.com/cruxctl/crux/internal/pty"
)

func (c *CLI) agentCommandUsage(agentName string, args []string) bool {
	if strings.TrimSpace(agentName) == "" {
		return false
	}
	specs, err := agent.LoadBuiltinSpecs()
	if err != nil {
		return false
	}
	if _, ok := agent.ResolveSpec(specs, agentName); !ok {
		return false
	}
	switch {
	case len(args) == 0:
		fmt.Fprintf(c.out, `Usage:
  crux %[1]s describe
  crux %[1]s usage
  crux %[1]s exec [--repo DIR] [-- PROVIDER_ARGS...]
  crux %[1]s conversations ls

Inspect and operate one discovered agent.
`, agentName)
	case args[0] == "describe":
		fmt.Fprintf(c.out, "Usage:\n  crux %s describe\n", agentName)
	case args[0] == "usage":
		fmt.Fprintf(c.out, "Usage:\n  crux %s usage\n", agentName)
	case args[0] == "exec":
		fmt.Fprintf(c.out, "Usage:\n  crux %s exec [--repo DIR] [--workdir DIR] [--timeout SECONDS] [-- PROVIDER_ARGS...]\n", agentName)
	case args[0] == "conversations":
		fmt.Fprintf(c.out, "Usage:\n  crux %s conversations ls\n", agentName)
	default:
		return false
	}
	return true
}

func (c *CLI) agentScoped(ctx context.Context, opts rootOptions, name string, args []string) error {
	specs, err := agent.LoadBuiltinSpecs()
	if err != nil {
		return err
	}
	spec, ok := agent.ResolveSpec(specs, name)
	if !ok {
		return fmt.Errorf("unknown command or agent %q", name)
	}
	if len(args) == 0 {
		args = []string{"describe"}
	}
	if helpArg(args) {
		c.agentCommandUsage(name, nil)
		return nil
	}
	if len(args) > 1 && (args[1] == "-h" || args[1] == "--help" || args[1] == "help") {
		c.agentCommandUsage(name, args[:1])
		return nil
	}
	store, err := agent.DefaultStore()
	if err != nil {
		return err
	}
	state, err := ensureAgentState(ctx, store, specs, spec)
	if err != nil {
		return err
	}
	switch args[0] {
	case "describe":
		if len(args) != 1 {
			return fmt.Errorf("usage: crux %s describe", name)
		}
		return c.describePTYAgent(ctx, opts, store, spec, state)
	case "usage":
		if len(args) != 1 {
			return fmt.Errorf("usage: crux %s usage", name)
		}
		return c.runPTYProbe(ctx, opts, store, spec, state, "usage")
	case "conversations":
		if len(args) == 0 || (len(args) == 1 && (args[0] == "-h" || args[0] == "--help")) {
			c.agentCommandUsage(name, []string{"conversations"})
			return nil
		}
		if len(args) != 2 || args[1] != "ls" {
			return fmt.Errorf("usage: crux %s conversations ls", name)
		}
		return c.runPTYProbe(ctx, opts, store, spec, state, "conversations_ls")
	case "exec":
		return c.execPTYAgent(ctx, opts, store, spec, state, args[1:])
	default:
		return fmt.Errorf("unknown agent command %q; expected describe, usage, exec, or conversations ls", args[0])
	}
}

func (c *CLI) discoverAgents(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: crux discover")
	}
	specs, err := agent.LoadBuiltinSpecs()
	if err != nil {
		return err
	}
	store, err := agent.DefaultStore()
	if err != nil {
		return err
	}
	results, err := agent.Discover(ctx, specs, pty.NewRunner(pty.NewFactory(), pty.NewNormalizer()))
	if err != nil {
		return err
	}
	states := make([]agent.AgentState, 0, len(results))
	for _, result := range results {
		if err := store.SaveAgent(result.State); err != nil {
			return err
		}
		states = append(states, result.State)
	}
	if opts.output != "table" {
		return c.print(opts.output, states)
	}
	fmt.Fprintln(c.out, "Discovered agents:")
	fmt.Fprintln(c.out)
	printAgentStateTable(c.out, states)
	return nil
}

func (c *CLI) psAgents(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: crux ps")
	}
	store, err := agent.DefaultStore()
	if err != nil {
		return err
	}
	states, err := store.ListAgents()
	if err != nil {
		return err
	}
	if len(states) == 0 {
		specs, loadErr := agent.LoadBuiltinSpecs()
		if loadErr != nil {
			return loadErr
		}
		results, discoverErr := agent.Discover(ctx, specs, pty.NewRunner(pty.NewFactory(), pty.NewNormalizer()))
		if discoverErr != nil {
			return discoverErr
		}
		for _, result := range results {
			if err := store.SaveAgent(result.State); err != nil {
				return err
			}
			states = append(states, result.State)
		}
	}
	if opts.output != "table" {
		return c.print(opts.output, states)
	}
	printAgentStateTable(c.out, states)
	return nil
}

func (c *CLI) describePTYAgent(ctx context.Context, opts rootOptions, store agent.Store, spec agent.Spec, state agent.AgentState) error {
	var probe *agent.ProbeResult
	if state.Available {
		result, err := runSpecProbe(ctx, store, spec, state, "describe", currentWorkingDir())
		if err == nil {
			probe = &result
			state.LastProbeAt = result.EndedAt
			_ = store.SaveAgent(state)
		}
	}
	if opts.output != "table" {
		return c.print(opts.output, struct {
			Agent agent.AgentState   `json:"agent" yaml:"agent"`
			Probe *agent.ProbeResult `json:"probe,omitempty" yaml:"probe,omitempty"`
		}{Agent: state, Probe: probe})
	}
	printAgentDescription(c.out, state, spec, probe)
	return nil
}

func (c *CLI) runPTYProbe(ctx context.Context, opts rootOptions, store agent.Store, spec agent.Spec, state agent.AgentState, probeName string) error {
	if !state.Available {
		return fmt.Errorf("agent %q is missing; run crux discover after installing %s", spec.ID, spec.Binary)
	}
	result, err := runSpecProbe(ctx, store, spec, state, probeName, currentWorkingDir())
	if err != nil {
		return err
	}
	if opts.output != "table" {
		return c.print(opts.output, result)
	}
	printProbeResult(c.out, result)
	return nil
}

func runSpecProbe(ctx context.Context, store agent.Store, spec agent.Spec, state agent.AgentState, probeName string, workDir string) (agent.ProbeResult, error) {
	task, probe, ok := agent.BuildProbeTask(spec, probeName, state.BinaryPath, workDir)
	if !ok {
		return agent.ProbeResult{}, fmt.Errorf("agent %q has no %s probe", spec.ID, probeName)
	}
	result, err := pty.NewRunner(pty.NewFactory(), pty.NewNormalizer()).Run(ctx, task)
	if err != nil {
		return agent.ProbeResult{}, err
	}
	sourceText := ""
	if result.Normalized != nil {
		switch task.ParseFrom {
		case "final_screen":
			sourceText = result.Normalized.FinalScreen
		default:
			sourceText = result.Normalized.CleanText
		}
	}
	parsed := agent.ParseProbeText(probe, sourceText)
	return store.SavePTYProbe(spec.ID, probeName, result, task.ParseFrom, parsed)
}

func (c *CLI) execPTYAgent(ctx context.Context, opts rootOptions, store agent.Store, spec agent.Spec, state agent.AgentState, args []string) error {
	if !state.Available {
		return fmt.Errorf("agent %q is missing; run crux discover after installing %s", spec.ID, spec.Binary)
	}
	execOpts, err := parsePTYExecArgs(args)
	if err != nil {
		return err
	}
	sessionID := newSessionID()
	command := spec.Launch.Command
	if command == "" || command == spec.Binary {
		command = state.BinaryPath
	}
	stdout := c.out
	if opts.output != "table" {
		stdout = c.err
	}
	task := pty.PTYTask{
		AgentName:   spec.ID,
		Purpose:     "exec",
		Command:     command,
		Args:        append(append([]string{}, spec.Launch.Args...), execOpts.ProviderArgs...),
		WorkDir:     execOpts.WorkDir,
		Env:         execEnv(spec),
		Normalize:   spec.Normalize,
		Timeout:     execOpts.Timeout,
		Interactive: true,
		Stdin:       os.Stdin,
		Stdout:      stdout,
		Stderr:      c.err,
	}
	if task.WorkDir == "" {
		task.WorkDir = currentWorkingDir()
	}
	result, err := pty.NewRunner(pty.NewFactory(), pty.NewNormalizer()).Run(ctx, task)
	if err != nil {
		return err
	}
	outputState, err := store.SaveSessionOutput(sessionID, result.Raw, result.Normalized)
	if err != nil {
		return err
	}
	session := agent.SessionState{
		ID:             sessionID,
		Agent:          spec.ID,
		Status:         result.Status,
		StartedAt:      result.StartedAt,
		EndedAt:        result.EndedAt,
		TranscriptRaw:  outputState.TranscriptRaw,
		TranscriptText: outputState.TranscriptText,
		Error:          result.Error,
	}
	if err := store.SaveSession(session); err != nil {
		return err
	}
	if opts.output != "table" {
		return c.print(opts.output, session)
	}
	fmt.Fprintf(c.out, "\nSession: %s status=%s\n", session.ID, session.Status)
	fmt.Fprintf(c.out, "Transcript: %s\n", session.TranscriptText)
	return nil
}

type ptyExecOptions struct {
	WorkDir      string
	Timeout      time.Duration
	ProviderArgs []string
}

func parsePTYExecArgs(args []string) (ptyExecOptions, error) {
	out := ptyExecOptions{Timeout: 24 * time.Hour}
	positionals := make([]string, 0)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			out.ProviderArgs = append(out.ProviderArgs, args[i+1:]...)
			return out, nil
		case arg == "--repo" || arg == "--workdir":
			i++
			if i >= len(args) {
				return out, fmt.Errorf("%s requires a directory", arg)
			}
			out.WorkDir = args[i]
		case strings.HasPrefix(arg, "--repo="):
			out.WorkDir = strings.TrimPrefix(arg, "--repo=")
		case strings.HasPrefix(arg, "--workdir="):
			out.WorkDir = strings.TrimPrefix(arg, "--workdir=")
		case arg == "--timeout":
			i++
			if i >= len(args) {
				return out, fmt.Errorf("--timeout requires seconds")
			}
			timeout, err := parseSeconds(args[i])
			if err != nil {
				return out, err
			}
			out.Timeout = timeout
		case strings.HasPrefix(arg, "--timeout="):
			timeout, err := parseSeconds(strings.TrimPrefix(arg, "--timeout="))
			if err != nil {
				return out, err
			}
			out.Timeout = timeout
		default:
			positionals = append(positionals, arg)
		}
	}
	if len(positionals) > 0 {
		return out, fmt.Errorf("usage: crux <agent> exec [--repo DIR] [-- PROVIDER_ARGS...]")
	}
	if out.WorkDir != "" {
		abs, err := filepath.Abs(out.WorkDir)
		if err != nil {
			return out, err
		}
		out.WorkDir = abs
	}
	return out, nil
}

func ensureAgentState(ctx context.Context, store agent.Store, specs []agent.Spec, spec agent.Spec) (agent.AgentState, error) {
	state, err := store.LoadAgent(spec.ID)
	if err == nil {
		return state, nil
	}
	results, err := agent.Discover(ctx, specs, pty.NewRunner(pty.NewFactory(), pty.NewNormalizer()))
	if err != nil {
		return agent.AgentState{}, err
	}
	for _, result := range results {
		if err := store.SaveAgent(result.State); err != nil {
			return agent.AgentState{}, err
		}
		if result.State.ID == spec.ID {
			state = result.State
		}
	}
	if state.ID == "" {
		return agent.AgentState{}, fmt.Errorf("agent %q not found", spec.ID)
	}
	return state, nil
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

func printAgentDescription(out io.Writer, state agent.AgentState, spec agent.Spec, probe *agent.ProbeResult) {
	fmt.Fprintf(out, "Agent: %s\n", state.Name)
	fmt.Fprintf(out, "Provider: %s\n", firstNonEmpty(state.Provider, "-"))
	fmt.Fprintf(out, "Binary: %s\n", firstNonEmpty(state.BinaryPath, state.Binary))
	fmt.Fprintf(out, "Version: %s\n", firstNonEmpty(state.Version, "-"))
	if len(state.ConfigPaths) > 0 {
		fmt.Fprintln(out, "Config paths:")
		for _, path := range state.ConfigPaths {
			fmt.Fprintf(out, "- %s\n", path)
		}
	}
	commands := spec.KnownCommands()
	if len(commands) > 0 {
		fmt.Fprintln(out, "Known TUI commands:")
		for _, name := range commands {
			probe := spec.Commands[name]
			fmt.Fprintf(out, "- %s", strings.ReplaceAll(name, "_", " "))
			if strings.TrimSpace(probe.Input) != "" {
				fmt.Fprintf(out, " (%s)", oneLine(probe.Input))
			}
			fmt.Fprintln(out)
		}
	}
	if !state.LastDiscoveredAt.IsZero() {
		fmt.Fprintf(out, "Last discovery: %s\n", state.LastDiscoveredAt.Format(time.RFC3339))
	}
	if probe != nil && probe.CleanPath != "" {
		fmt.Fprintf(out, "Last describe probe: %s confidence=%s\n", probe.CleanPath, probe.Confidence)
	}
}

func printProbeResult(out io.Writer, result agent.ProbeResult) {
	fmt.Fprintf(out, "Agent: %s\n", result.AgentName)
	fmt.Fprintf(out, "Probe: %s\n", result.ProbeName)
	fmt.Fprintf(out, "Usage source: PTY probe\n")
	fmt.Fprintf(out, "Last updated: %s\n", result.EndedAt.Format(time.RFC3339))
	fmt.Fprintf(out, "Confidence: %s\n", firstNonEmpty(result.Confidence, "low"))
	if len(result.Parsed) > 0 {
		fmt.Fprintln(out, "Parsed:")
		keys := make([]string, 0, len(result.Parsed))
		for key := range result.Parsed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(out, "  %s: %s\n", key, result.Parsed[key])
		}
	}
	text := result.CleanText
	if result.ParseSource == "final_screen" && result.FinalScreen != "" {
		text = result.FinalScreen
	}
	if strings.TrimSpace(text) != "" {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Clean output:")
		fmt.Fprintln(out, text)
	}
	if result.Error != "" {
		fmt.Fprintf(out, "Error: %s\n", result.Error)
	}
}

func execEnv(spec agent.Spec) map[string]string {
	env := map[string]string{
		"CRUX_PTY_MODE": "interactive",
		"CRUX_AGENT_ID": spec.ID,
	}
	for key, value := range spec.PTYEnv {
		env[key] = value
	}
	return env
}

func parseSeconds(value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err == nil {
		return duration, nil
	}
	var seconds int
	if _, scanErr := fmt.Sscanf(value, "%d", &seconds); scanErr != nil || seconds < 1 {
		return 0, fmt.Errorf("--timeout must be a positive duration or seconds value")
	}
	return time.Duration(seconds) * time.Second, nil
}

func newSessionID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "sess_" + time.Now().UTC().Format("20060102150405")
	}
	return "sess_" + hex.EncodeToString(buf[:])
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
