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
	command := spec.Detect.Command
	if command == "" || command == spec.Binary {
		command = path
	}
	if runner == nil {
		return ""
	}
	result, err := runner.Run(ctx, pty.PTYTask{
		AgentName:     spec.ID,
		Purpose:       "detect",
		Command:       command,
		Args:          spec.Detect.Args,
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
	command := spec.Launch.Command
	if command == "" || command == spec.Binary {
		command = binaryPath
	}
	task := pty.PTYTask{
		AgentName:     spec.ID,
		Purpose:       probeName,
		Command:       command,
		Args:          append([]string{}, spec.Launch.Args...),
		WorkDir:       workDir,
		Env:           probeEnv(spec, probeName),
		Input:         probe.Input,
		ReadyMatcher:  spec.Ready,
		DoneMatcher:   probe.CompleteWhen,
		Normalize:     spec.Normalize,
		ParseFrom:     firstNonEmpty(probe.ParseFrom, "clean_text"),
		Timeout:       ProbeTimeout(probe, 20*time.Second),
		CaptureOutput: true,
	}
	return task, probe, true
}
