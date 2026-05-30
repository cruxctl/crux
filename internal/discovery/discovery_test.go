package discovery

import (
	"context"
	"testing"

	"github.com/cruxctl/crux/internal/agent"
	"github.com/cruxctl/crux/pkg/cruxapi"
)

func TestDiscovererFindsNothingWithEmptyRegistry(t *testing.T) {
	reg := agent.NewRegistry()
	d := NewDiscoverer(reg)
	res, err := d.Discover(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Instances) != 0 {
		t.Fatalf("expected no instances, got %d", len(res.Instances))
	}
}

func TestDiscovererRespectsAgentFilter(t *testing.T) {
	reg := agent.NewRegistry()
	reg.Set(cruxapi.CodingAgentSpec{
		APIVersion: "crux.dev/v1alpha1",
		Kind:       "CodingAgentSpec",
		Metadata:   cruxapi.AgentSpecMetadata{ID: "test-agent", Name: "Test"},
		Detection: cruxapi.AgentSpecDetection{
			Binaries: []string{"this-binary-definitely-does-not-exist-12345"},
		},
	})
	d := NewDiscoverer(reg)
	_, err := d.Discover(context.Background(), "unknown-agent")
	if err == nil {
		t.Fatal("expected error for unknown agent filter")
	}
	res, err := d.Discover(context.Background(), "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Instances) != 0 {
		t.Fatalf("expected no instances for missing binary, got %d", len(res.Instances))
	}
}

func TestExpandTilde(t *testing.T) {
	if got := expandTilde("/absolute/path"); got != "/absolute/path" {
		t.Fatalf("expected /absolute/path, got %s", got)
	}
}

func TestUniqueStrings(t *testing.T) {
	in := []string{"a", "b", "a", "c", "b"}
	got := uniqueStrings(in)
	if len(got) != 3 {
		t.Fatalf("expected 3 unique, got %d", len(got))
	}
}
