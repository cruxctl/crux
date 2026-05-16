package discovery

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/cruxctl/crux/internal/domain"
)

type Discoverer struct {
	Candidates []Candidate
}

type Candidate struct {
	Name        string
	Description string
	VersionArgs []string
	RunArgs     []string
}

type Result struct {
	Agent   domain.Agent `json:"agent"`
	Version string       `json:"version"`
}

func DefaultDiscoverer() Discoverer {
	return Discoverer{
		Candidates: []Candidate{
			{
				Name:        "claude",
				Description: "Claude Code managed CLI agent",
				VersionArgs: []string{"--version"},
				RunArgs:     []string{"-p", "{prompt}"},
			},
			{
				Name:        "codex",
				Description: "OpenAI Codex managed CLI agent",
				VersionArgs: []string{"--version"},
				RunArgs:     []string{"exec", "{prompt}"},
			},
			{
				Name:        "gemini",
				Description: "Gemini CLI managed agent",
				VersionArgs: []string{"--version"},
				RunArgs:     []string{"-p", "{prompt}"},
			},
			{
				Name:        "kimi",
				Description: "Kimi managed CLI agent",
				VersionArgs: []string{"--version"},
				RunArgs:     []string{"-p", "{prompt}"},
			},
		},
	}
}

func (d Discoverer) Discover(ctx context.Context, timeoutSeconds int) ([]Result, error) {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	results := make([]Result, 0, len(d.Candidates))
	for _, candidate := range d.Candidates {
		path, err := exec.LookPath(candidate.Name)
		if err != nil {
			continue
		}
		version := readVersion(ctx, timeout, path, candidate.VersionArgs)
		now := domain.Now()
		agent := domain.Agent{
			ID:          domain.NewID("agent"),
			Name:        candidate.Name,
			Description: candidate.Description,
			Labels: map[string]string{
				"cruxctl.io/class":  "managed-cli",
				"cruxctl.io/source": "path-discovery",
			},
			Command: domain.CommandSpec{
				Path: path,
				Args: candidate.RunArgs,
			},
			Status:    domain.AgentReady,
			CreatedAt: now,
			UpdatedAt: now,
		}
		results = append(results, Result{Agent: agent, Version: version})
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func readVersion(parent context.Context, timeout time.Duration, path string, args []string) string {
	if len(args) == 0 {
		return ""
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, args...).CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
