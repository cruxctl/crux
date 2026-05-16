package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cruxctl/crux/internal/domain"
)

func TestFileStorePersistsAgentsExecutionsAndEvents(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state.json")
	runtime := domain.DefaultRuntimeConfig()
	st, err := NewFileStore(path, runtime)
	if err != nil {
		t.Fatal(err)
	}
	agent := domain.Agent{
		Name: "Echo",
		Command: domain.CommandSpec{
			Path: "/usr/bin/printf",
			Args: []string{"%s", "{prompt}"},
		},
	}
	if err := st.UpsertAgent(ctx, agent); err != nil {
		t.Fatal(err)
	}
	gotAgent, err := st.GetAgent(ctx, "echo")
	if err != nil {
		t.Fatal(err)
	}
	if gotAgent.Name != "echo" {
		t.Fatalf("expected normalized name echo, got %s", gotAgent.Name)
	}

	execution := domain.Execution{
		ID:            domain.NewID("exec"),
		AgentName:     "echo",
		Status:        domain.ExecutionQueued,
		QueuedAt:      domain.Now(),
		UpdatedAt:     domain.Now(),
		RuntimeConfig: runtime,
	}
	if err := st.CreateExecution(ctx, execution); err != nil {
		t.Fatal(err)
	}
	if err := st.AppendEvent(ctx, domain.Event{Type: domain.EventExecutionQueued, ExecutionID: execution.ID, Message: "queued"}); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewFileStore(path, runtime)
	if err != nil {
		t.Fatal(err)
	}
	events, err := reopened.ListEvents(ctx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}
