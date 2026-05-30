package runner

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

func TestCommandRunnerInjectsPromptArg(t *testing.T) {
	run := NewCommandRunner()
	result := run.Run(context.Background(), cruxapi.Agent{
		Name: "echo",
		Command: cruxapi.CommandSpec{
			Path: "/usr/bin/printf",
			Args: []string{"%s", "{prompt}"},
		},
	}, cruxapi.Execution{Prompt: "hello"}, cruxapi.DefaultRuntimeConfig())
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if result.Stdout != "hello" {
		t.Fatalf("expected hello, got %q", result.Stdout)
	}
}

func TestCommandRunnerBlocksShellByDefault(t *testing.T) {
	run := NewCommandRunner()
	result := run.Run(context.Background(), cruxapi.Agent{
		Name: "shell",
		Command: cruxapi.CommandSpec{
			Path: filepath.Join("/bin", "sh"),
			Args: []string{"-c", "echo unsafe"},
		},
	}, cruxapi.Execution{Prompt: "hello"}, cruxapi.DefaultRuntimeConfig())
	if result.Error == "" {
		t.Fatal("expected shell command to be rejected")
	}
}

func TestCommandRunnerUsesExecutionWorkingDir(t *testing.T) {
	run := NewCommandRunner()
	workDir := t.TempDir()
	result := run.Run(context.Background(), cruxapi.Agent{
		Name: "pwd",
		Command: cruxapi.CommandSpec{
			Path: "/usr/bin/pwd",
		},
	}, cruxapi.Execution{WorkingDir: workDir}, cruxapi.DefaultRuntimeConfig())
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if result.Stdout != workDir+"\n" {
		t.Fatalf("expected working dir %q, got %q", workDir, result.Stdout)
	}
}

func TestCommandRunnerAgentWorkingDirOverridesExecutionWorkingDir(t *testing.T) {
	run := NewCommandRunner()
	agentDir := t.TempDir()
	requestDir := t.TempDir()
	result := run.Run(context.Background(), cruxapi.Agent{
		Name: "pwd",
		Command: cruxapi.CommandSpec{
			Path:       "/usr/bin/pwd",
			WorkingDir: agentDir,
		},
	}, cruxapi.Execution{WorkingDir: requestDir}, cruxapi.DefaultRuntimeConfig())
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if result.Stdout != agentDir+"\n" {
		t.Fatalf("expected agent working dir %q, got %q", agentDir, result.Stdout)
	}
}
