package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cruxctl/crux/internal/discovery"
	"github.com/cruxctl/crux/internal/domain"
	"github.com/cruxctl/crux/internal/runner"
	"github.com/cruxctl/crux/internal/store"
)

func TestServiceRunsExecutionAndCapturesEvents(t *testing.T) {
	ctx := context.Background()
	runtime := domain.DefaultRuntimeConfig()
	st, err := store.NewFileStore(filepath.Join(t.TempDir(), "state.json"), runtime)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, runner.NewCommandRunner(), discovery.DefaultDiscoverer(), runtime, nil)
	_, err = svc.UpsertAgent(ctx, domain.Agent{
		Name: "echo",
		Command: domain.CommandSpec{
			Path: "/usr/bin/printf",
			Args: []string{"%s", "{prompt}"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := svc.SubmitExecution(ctx, SubmitRequest{AgentName: "echo", Prompt: "ok", Wait: true})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != domain.ExecutionSucceeded {
		t.Fatalf("expected succeeded, got %s: %s", execution.Status, execution.Error)
	}
	if execution.Stdout != "ok" {
		t.Fatalf("expected stdout ok, got %q", execution.Stdout)
	}
	events, err := svc.ListEvents(ctx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 3 {
		t.Fatalf("expected at least 3 execution events, got %d", len(events))
	}
}
