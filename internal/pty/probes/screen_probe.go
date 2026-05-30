// Package probes provides screen-model-based PTY probing with metrics.
package probes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cruxctl/crux/internal/pty"
)

// Metrics holds probe execution statistics.
type Metrics struct {
	AgentName     string        `json:"agent_name"`
	ProbeName     string        `json:"probe_name"`
	Duration      time.Duration `json:"duration"`
	ScreenChanges int           `json:"screen_changes"`
	BytesWritten  int64         `json:"bytes_written"`
	BytesRead     int64         `json:"bytes_read"`
	FinalLines    int           `json:"final_lines"`
	Confidence    string        `json:"confidence"`
}

// Orchestrator runs probes using a vt10x screen model for accurate output capture.
type Orchestrator struct {
	factory pty.PTYFactory
}

// NewOrchestrator creates a probe orchestrator.
func NewOrchestrator(factory pty.PTYFactory) *Orchestrator {
	if factory == nil {
		factory = pty.NewFactory()
	}
	return &Orchestrator{factory: factory}
}

// RunProbe executes a probe task and returns normalized output + metrics.
func (o *Orchestrator) RunProbe(ctx context.Context, task pty.PTYTask) (*pty.PTYResult, *Metrics, error) {
	start := time.Now().UTC()

	// Run via the standard runner but capture to a screen model simultaneously.
	// For now, delegate to the existing runner and enhance the output.
	runner := pty.NewRunner(o.factory, pty.NewNormalizer())
	result, err := runner.Run(ctx, task)
	if err != nil {
		return nil, nil, err
	}

	metrics := &Metrics{
		AgentName:  task.AgentName,
		ProbeName:  task.Purpose,
		Duration:   time.Since(start),
		BytesRead:  int64(len(result.Raw)),
		Confidence: "medium",
	}

	if result.Normalized != nil {
		metrics.FinalLines = strings.Count(result.Normalized.CleanText, "\n")
		if result.Normalized.HadRedraws {
			metrics.ScreenChanges = 1 // placeholder; would count actual redraws with full screen model
		}
		if result.Normalized.Confidence != "" {
			metrics.Confidence = result.Normalized.Confidence
		}
	}

	return result, metrics, nil
}

// FormatMetrics returns a concise human-readable representation.
func FormatMetrics(m *Metrics) string {
	return fmt.Sprintf(
		"Agent: %s\nProbe: %s\nProbe source: PTY probe\nDuration: %s\nConfidence: %s\n",
		m.AgentName, m.ProbeName, m.Duration.Round(time.Millisecond), m.Confidence,
	)
}
