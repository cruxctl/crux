package agent

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/cruxctl/crux/internal/pty"
)

type DiscoveryResult struct {
	State AgentState `json:"agent" yaml:"agent"`
	Spec  Spec       `json:"-" yaml:"-"`
}

func Discover(ctx context.Context, specs []Spec, runner pty.PTYRunner) ([]DiscoveryResult, error) {
	results := make([]DiscoveryResult, 0, len(specs))
	now := time.Now().UTC()
	for _, spec := range specs {
		path, err := exec.LookPath(spec.Binary)
		state := AgentState{
			ID:               spec.ID,
			Name:             spec.Name,
			Provider:         spec.Provider,
			Binary:           spec.Binary,
			Available:        err == nil,
			Status:           "missing",
			ConfigPaths:      spec.ConfigPaths,
			KnownCommands:    spec.KnownCommands(),
			LastDiscoveredAt: now,
		}
		if err == nil {
			state.BinaryPath = path
			state.Status = "available"
			state.Version = detectVersion(ctx, spec, path, runner)
		}
		results = append(results, DiscoveryResult{State: state, Spec: spec})
	}
	return results, ctx.Err()
}

func detectVersion(ctx context.Context, spec Spec, path string, runner pty.PTYRunner) string {
	if runner == nil {
		return ""
	}
	result, err := runner.Run(ctx, pty.PTYTask{
		AgentName:     spec.ID,
		Purpose:       "detect",
		Command:       ResolveCommand(spec.Detect.Command, spec.Binary, path, ""),
		Args:          ExpandArgs(spec.Detect.Args, path, ""),
		Env:           probeEnv(spec, "detect"),
		Normalize:     spec.Normalize,
		Timeout:       5 * time.Second,
		CaptureOutput: true,
	})
	if err != nil || result == nil {
		return ""
	}
	if result.Normalized != nil {
		return firstLine(result.Normalized.CleanText)
	}
	return firstLine(result.Text)
}

func probeEnv(spec Spec, probeName string) map[string]string {
	env := map[string]string{
		"CRUX_PTY_MODE": "probe",
		"CRUX_AGENT_ID": spec.ID,
		"CRUX_PROBE_ID": probeName,
	}
	for key, value := range spec.PTYEnv {
		env[key] = value
	}
	return env
}

func firstLine(value string) string {
	for _, line := range strings.Split(value, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func BuildProbeTask(spec Spec, probeName string, binaryPath string, workDir string) (pty.PTYTask, CommandProbe, bool) {
	probe, ok := spec.Commands[probeName]
	if !ok {
		return pty.PTYTask{}, CommandProbe{}, false
	}
	workDir = ExpandValue(firstNonEmpty(probe.WorkDir, workDir), binaryPath, workDir)
	command := spec.Launch.Command
	args := append([]string{}, spec.Launch.Args...)
	if strings.TrimSpace(probe.Command) != "" {
		command = probe.Command
		args = nil
	}
	if len(probe.Args) > 0 {
		args = append(args, probe.Args...)
	}
	ready := spec.Ready
	if strings.TrimSpace(probe.Ready.Strategy) != "" {
		ready = probe.Ready
	}
	if strings.TrimSpace(probe.Input) == "" {
		ready = pty.MatcherSpec{}
	}
	task := pty.PTYTask{
		AgentName:     spec.ID,
		Purpose:       probeName,
		Command:       ResolveCommand(command, spec.Binary, binaryPath, workDir),
		Args:          ExpandArgs(args, binaryPath, workDir),
		WorkDir:       workDir,
		Env:           probeEnv(spec, probeName),
		Input:         probe.Input,
		ReadyMatcher:  ready,
		DoneMatcher:   probe.CompleteWhen,
		Normalize:     spec.Normalize,
		ParseFrom:     firstNonEmpty(probe.ParseFrom, "clean_text"),
		Timeout:       ProbeTimeout(probe, 20*time.Second),
		CaptureOutput: true,
	}
	return task, probe, true
}
