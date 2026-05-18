package agent

import (
	"strings"
	"testing"

	"github.com/cruxctl/crux/internal/pty"
)

func TestBuildProbeTaskUsesCommandOverride(t *testing.T) {
	spec := Spec{
		ID:     "kimi-cli",
		Name:   "Kimi CLI",
		Binary: "kimi",
		Launch: CommandSpec{
			Command: "kimi",
			Args:    []string{"--work-dir", "{workdir}"},
		},
		Commands: map[string]CommandProbe{
			"usage": {
				Command: "kimi",
				Args:    []string{"info", "--json"},
			},
		},
	}
	spec.ApplyDefaults()

	task, _, ok := BuildProbeTask(spec, "usage", "/usr/local/bin/kimi", "/repo")
	if !ok {
		t.Fatal("expected usage probe")
	}
	if task.Command != "/usr/local/bin/kimi" {
		t.Fatalf("expected binary path command, got %q", task.Command)
	}
	if strings.Join(task.Args, " ") != "info --json" {
		t.Fatalf("unexpected args: %#v", task.Args)
	}
	if task.ReadyMatcher.Strategy != "" {
		t.Fatalf("non-input command probes should not wait for TUI readiness: %#v", task.ReadyMatcher)
	}
}

func TestBuildProbeTaskExpandsLaunchArgsForTUIProbe(t *testing.T) {
	spec := Spec{
		ID:     "kimi-cli",
		Name:   "Kimi CLI",
		Binary: "kimi",
		Launch: CommandSpec{
			Command: "kimi",
			Args:    []string{"--work-dir", "{workdir}"},
		},
		Ready: ptyMatcher("screen_contains", "ready"),
		Commands: map[string]CommandProbe{
			"conversations_ls": {
				Input: "/sessions\n",
			},
		},
	}
	spec.ApplyDefaults()

	task, _, ok := BuildProbeTask(spec, "conversations_ls", "/usr/local/bin/kimi", "/repo")
	if !ok {
		t.Fatal("expected conversations probe")
	}
	if strings.Join(task.Args, " ") != "--work-dir /repo" {
		t.Fatalf("unexpected launch args: %#v", task.Args)
	}
	if task.ReadyMatcher.Strategy != "screen_contains" {
		t.Fatalf("expected inherited readiness matcher, got %#v", task.ReadyMatcher)
	}
}

func ptyMatcher(strategy string, pattern string) pty.MatcherSpec {
	return pty.MatcherSpec{Strategy: strategy, Pattern: pattern}
}
