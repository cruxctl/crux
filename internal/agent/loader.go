// Package agents loads, validates, and indexes CodingAgentSpec YAML files
// from disk. Specs live in two places:
//
//   1. Daemon defaults: cruxd/examples/agents/*.yaml (read-only).
//   2. User overrides:  ~/.crux/state/agents/specs/*.yaml (writable).
//
// Conflict rule: same metadata.id in both → user override wins.
package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

// LoadDir reads every *.yaml file in dir and returns the valid specs.
// Returns a multi-error containing parse and validate failures (other
// specs still load).
func LoadDir(dir string) ([]cruxapi.CodingAgentSpec, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var specs []cruxapi.CodingAgentSpec
	var firstErr error
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		spec, err := LoadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", e.Name(), err)
			}
			continue
		}
		specs = append(specs, spec)
	}
	return specs, firstErr
}

func LoadFile(path string) (cruxapi.CodingAgentSpec, error) {
	var s cruxapi.CodingAgentSpec
	body, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	if err := yaml.Unmarshal(body, &s); err != nil {
		return s, err
	}
	if err := Validate(s); err != nil {
		return s, err
	}
	return s, nil
}

// Registry holds loaded specs indexed by metadata.id.
type Registry struct {
	byID map[string]cruxapi.CodingAgentSpec
}

func NewRegistry() *Registry { return &Registry{byID: map[string]cruxapi.CodingAgentSpec{}} }

func (r *Registry) Set(spec cruxapi.CodingAgentSpec) { r.byID[spec.Metadata.ID] = spec }

func (r *Registry) Get(id string) (cruxapi.CodingAgentSpec, bool) {
	s, ok := r.byID[id]
	return s, ok
}

func (r *Registry) All() []cruxapi.CodingAgentSpec {
	out := make([]cruxapi.CodingAgentSpec, 0, len(r.byID))
	for _, v := range r.byID {
		out = append(out, v)
	}
	return out
}

// LoadAll loads daemon defaults then user overrides into a Registry.
// userDir may be empty to skip user overrides.
func LoadAll(defaultsDir, userDir string) (*Registry, error) {
	reg := NewRegistry()
	specs, dErr := LoadDir(defaultsDir)
	for _, s := range specs {
		reg.Set(s)
	}
	var uErr error
	if userDir != "" {
		if _, err := os.Stat(userDir); err == nil {
			us, e := LoadDir(userDir)
			for _, s := range us {
				reg.Set(s)
			}
			uErr = e
		}
	}
	switch {
	case dErr != nil && uErr != nil:
		return reg, errors.New(dErr.Error() + "; " + uErr.Error())
	case dErr != nil:
		return reg, dErr
	case uErr != nil:
		return reg, uErr
	}
	return reg, nil
}
