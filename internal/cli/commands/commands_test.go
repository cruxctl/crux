package commands

import (
	"context"
	"testing"
)

func TestRootCommands_PresentInOrder(t *testing.T) {
	got := Root()
	want := []string{
		"version", "doctor", "update", "discover", "daemon", "context", "config",
		"agents", "agent", "run", "ps", "sessions", "trace", "events",
		"gateway", "mcp", "policy", "aos", "agbom", "console",
		"usage", "machines", "audit",
	}
	if len(got.Subcommands) != len(want) {
		t.Fatalf("len = %d, want %d", len(got.Subcommands), len(want))
	}
	for i, name := range want {
		if got.Subcommands[i].Name != name {
			t.Errorf("Subcommands[%d] = %s, want %s", i, got.Subcommands[i].Name, name)
		}
	}
	_ = context.Background()
}
