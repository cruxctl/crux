package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cruxctl/crux/internal/agent"
	"github.com/cruxctl/crux/internal/envpath"
	"github.com/cruxctl/crux/internal/statepath"
	"github.com/cruxctl/crux/pkg/cruxapi"
)

// Discoverer scans the system for installed coding agents using the specs
// registered in the agent Registry.
type Discoverer struct {
	registry *agent.Registry
}

// NewDiscoverer creates a Discoverer backed by the given Registry.
func NewDiscoverer(registry *agent.Registry) *Discoverer {
	return &Discoverer{registry: registry}
}

// AgentInstance is a discovered, running or installed agent with rich metadata.
type AgentInstance struct {
	SpecID      string            `json:"spec_id"`
	Name        string            `json:"name"`
	Provider    string            `json:"provider"`
	BinaryPath  string            `json:"binary_path,omitempty"`
	Version     string            `json:"version,omitempty"`
	ConfigFound []string          `json:"config_found,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	DiscoveredAt time.Time        `json:"discovered_at"`
	Source      string            `json:"source"`
}

// Result is the enriched output of a discovery run.
type Result struct {
	Instances []AgentInstance `json:"instances"`
	Errors    []string        `json:"errors,omitempty"`
	StartedAt time.Time       `json:"started_at"`
	EndedAt   time.Time       `json:"ended_at"`
}

// Discover runs a full discovery scan. If agentFilter is non-empty, only that
// agent spec is checked.
func (d *Discoverer) Discover(ctx context.Context, agentFilter string) (Result, error) {
	start := time.Now().UTC()
	result := Result{StartedAt: start, Instances: []AgentInstance{}}

	specs := d.registry.All()
	if agentFilter != "" {
		spec, ok := d.registry.Get(agentFilter)
		if !ok {
			return result, fmt.Errorf("unknown agent spec: %s", agentFilter)
		}
		specs = []cruxapi.CodingAgentSpec{spec}
	}

	for _, spec := range specs {
		if ctx.Err() != nil {
			break
		}
		inst, err := d.discoverOne(ctx, spec)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", spec.Metadata.ID, err))
			continue
		}
		if inst.BinaryPath != "" {
			result.Instances = append(result.Instances, inst)
		}
	}

	result.EndedAt = time.Now().UTC()
	return result, ctx.Err()
}

func (d *Discoverer) discoverOne(ctx context.Context, spec cruxapi.CodingAgentSpec) (AgentInstance, error) {
	inst := AgentInstance{
		SpecID:       spec.Metadata.ID,
		Name:         spec.Metadata.Name,
		Provider:     spec.Metadata.Provider,
		DiscoveredAt: time.Now().UTC(),
		Source:       "path_scan",
		Labels: map[string]string{
			"cruxctl.io/class":  "managed-cli",
			"cruxctl.io/source": "path-discovery",
		},
	}

	// 1. Find binary on PATH
	binaryPath := ""
	for _, name := range spec.Detection.Binaries {
		path, err := envpath.Lookup(name)
		if err == nil {
			binaryPath = path
			break
		}
	}
	if binaryPath == "" {
		// Not installed — return empty instance (caller skips)
		return inst, nil
	}
	inst.BinaryPath = binaryPath
	inst.Labels["cruxctl.io/binary"] = filepath.Base(binaryPath)

	// 2. Run version probe
	inst.Version = d.probeVersion(ctx, spec, binaryPath)
	inst.Labels["cruxctl.io/version"] = inst.Version

	// 3. Scan config paths
	inst.ConfigFound = d.scanConfigPaths(spec.Detection.ConfigPaths)

	return inst, nil
}

func (d *Discoverer) probeVersion(ctx context.Context, spec cruxapi.CodingAgentSpec, binaryPath string) string {
	vp := spec.Detection.Version
	if vp.Command == "" && len(vp.Args) == 0 {
		return ""
	}

	cmdPath := vp.Command
	if cmdPath == "" || cmdPath == spec.Metadata.ID || cmdPath == spec.Detection.Binaries[0] {
		cmdPath = binaryPath
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdPath, vp.Args...)
	cmd.Env = envpath.CommandEnv(os.Environ(), cmdPath, nil)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (d *Discoverer) scanConfigPaths(patterns []string) []string {
	found := []string{}
	for _, pattern := range patterns {
		expanded := expandTilde(pattern)
		matches, err := filepath.Glob(expanded)
		if err != nil {
			continue
		}
		for _, m := range matches {
			if _, err := os.Stat(m); err == nil {
				found = append(found, m)
			}
		}
		// Also try as literal path (not glob)
		if len(matches) == 0 {
			if _, err := os.Stat(expanded); err == nil {
				found = append(found, expanded)
			}
		}
	}
	return uniqueStrings(found)
}

// WriteResult persists a discovery result to ~/.crux/state/agents/<sha>/discovery.json
// and updates the agents index.
func WriteResult(result Result) error {
	if err := statepath.EnsureDir(statepath.IndexesDir()); err != nil {
		return err
	}

	// Write per-agent discovery files
	for _, inst := range result.Instances {
		dir := statepath.AgentDir(statepath.ID(inst.SpecID))
		if err := statepath.EnsureDir(dir); err != nil {
			return err
		}
		path := filepath.Join(dir, "discovery.json")
		data, err := json.MarshalIndent(inst, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return err
		}
	}

	// Write summary index
	indexPath := filepath.Join(statepath.IndexesDir(), "agents.index.json")
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(indexPath, data, 0o600)
}

func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~/") && path != "~" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/"))
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
