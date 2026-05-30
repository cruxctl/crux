package agent

import (
	"errors"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

var validInjectStrategies = map[string]bool{
	"write_config_file": true,
	"env_var":           true,
	"cli_command":       true,
	"tui_command":       true,
	"unsupported":       true,
	"":                  true, // absent is OK
}

// Validate enforces the minimum shape required by §6.1.
func Validate(s cruxapi.CodingAgentSpec) error {
	if s.APIVersion != "crux.dev/v1alpha1" {
		return errors.New("apiVersion must be crux.dev/v1alpha1")
	}
	if s.Kind != "CodingAgentSpec" {
		return errors.New("kind must be CodingAgentSpec")
	}
	if s.Metadata.ID == "" {
		return errors.New("metadata.id is required")
	}
	if s.Metadata.Name == "" {
		return errors.New("metadata.name is required")
	}
	if s.Metadata.Provider == "" {
		return errors.New("metadata.provider is required")
	}
	if !validInjectStrategies[s.MCPInject.Strategy] {
		return errors.New("mcp_inject.strategy must be one of write_config_file|env_var|cli_command|tui_command|unsupported")
	}
	return nil
}
