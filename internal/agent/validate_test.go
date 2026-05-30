package agent

import (
	"testing"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

func TestValidate_RequiresMetadataID(t *testing.T) {
	s := cruxapi.CodingAgentSpec{APIVersion: "crux.dev/v1alpha1", Kind: "CodingAgentSpec"}
	if err := Validate(s); err == nil {
		t.Errorf("validate should reject empty metadata.id")
	}
}

func TestValidate_RejectsUnknownMCPInjectStrategy(t *testing.T) {
	s := cruxapi.CodingAgentSpec{
		APIVersion: "crux.dev/v1alpha1",
		Kind:       "CodingAgentSpec",
		Metadata:   cruxapi.AgentSpecMetadata{ID: "x", Name: "X", Provider: "p"},
		MCPInject:  cruxapi.AgentMCPInject{Strategy: "telepathy"},
	}
	if err := Validate(s); err == nil {
		t.Errorf("validate should reject unknown strategy")
	}
}

func TestValidate_AcceptsBlueprintAgentSpec(t *testing.T) {
	s := cruxapi.CodingAgentSpec{
		APIVersion: "crux.dev/v1alpha1",
		Kind:       "CodingAgentSpec",
		Metadata:   cruxapi.AgentSpecMetadata{ID: "claude-code", Name: "Claude Code", Provider: "anthropic"},
		Detection:  cruxapi.AgentSpecDetection{Binaries: []string{"claude"}},
		Launch:     cruxapi.AgentSpecLaunch{Interactive: cruxapi.AgentLaunchMode{Command: "claude", RequiresPTY: true}},
		MCPInject:  cruxapi.AgentMCPInject{Strategy: "write_config_file"},
	}
	if err := Validate(s); err != nil {
		t.Errorf("expected valid: %v", err)
	}
}
