package runner

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cruxctl/crux/internal/domain"
)

func TestCommandRunnerInjectsPromptArg(t *testing.T) {
	run := NewCommandRunner()
	result := run.Run(context.Background(), domain.Agent{
		Name: "echo",
		Command: domain.CommandSpec{
			Path: "/usr/bin/printf",
			Args: []string{"%s", "{prompt}"},
		},
	}, domain.Execution{Prompt: "hello"}, domain.DefaultRuntimeConfig())
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if result.Stdout != "hello" {
		t.Fatalf("expected hello, got %q", result.Stdout)
	}
}

func TestCommandRunnerBlocksShellByDefault(t *testing.T) {
	run := NewCommandRunner()
	result := run.Run(context.Background(), domain.Agent{
		Name: "shell",
		Command: domain.CommandSpec{
			Path: filepath.Join("/bin", "sh"),
			Args: []string{"-c", "echo unsafe"},
		},
	}, domain.Execution{Prompt: "hello"}, domain.DefaultRuntimeConfig())
	if result.Error == "" {
		t.Fatal("expected shell command to be rejected")
	}
}
