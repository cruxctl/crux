package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/cruxctl/crux/internal/agent"
	"github.com/cruxctl/crux/internal/discovery"
	"github.com/cruxctl/crux/internal/runner"
	"github.com/cruxctl/crux/internal/state"
	"github.com/cruxctl/crux/pkg/cruxapi"
)

func testDiscoverer() *discovery.Discoverer {
	return discovery.NewDiscoverer(agent.NewRegistry())
}

func TestServiceRunsExecutionAndCapturesEvents(t *testing.T) {
	ctx := context.Background()
	runtime := cruxapi.DefaultRuntimeConfig()
	st, err := store.NewFileStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(store.NewDomainStore(st), runner.NewCommandRunner(), testDiscoverer(), runtime, nil)
	_, err = svc.UpsertAgent(ctx, cruxapi.Agent{
		Name: "echo",
		Command: cruxapi.CommandSpec{
			Path: "/usr/bin/printf",
			Args: []string{"%s", "{prompt}"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := svc.SubmitExecution(ctx, cruxapi.SubmitExecutionRequest{AgentName: "echo", Prompt: "ok", Wait: true})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != cruxapi.ExecutionSucceeded {
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
	usage, err := svc.AgentUsage(ctx, "echo")
	if err != nil {
		t.Fatal(err)
	}
	if usage.ExecutionsTotal != 1 || usage.Succeeded != 1 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
	if usage.StdoutBytes != 2 {
		t.Fatalf("expected stdout bytes 2, got %d", usage.StdoutBytes)
	}
	if usage.LastStdout != "ok" {
		t.Fatalf("expected last stdout ok, got %q", usage.LastStdout)
	}
}

func TestServiceStoresAndUsesRequestWorkingDir(t *testing.T) {
	ctx := context.Background()
	runtime := cruxapi.DefaultRuntimeConfig()
	st, err := store.NewFileStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(store.NewDomainStore(st), runner.NewCommandRunner(), testDiscoverer(), runtime, nil)
	_, err = svc.UpsertAgent(ctx, cruxapi.Agent{
		Name: "pwd",
		Command: cruxapi.CommandSpec{
			Path: "/usr/bin/pwd",
		},
		Status: cruxapi.AgentReady,
	})
	if err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	execution, err := svc.SubmitExecution(ctx, cruxapi.SubmitExecutionRequest{
		AgentName:  "pwd",
		WorkingDir: workDir,
		Wait:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if execution.WorkingDir != workDir {
		t.Fatalf("expected stored working dir %q, got %q", workDir, execution.WorkingDir)
	}
	if execution.Stdout != workDir+"\n" {
		t.Fatalf("expected pwd output %q, got %q", workDir+"\n", execution.Stdout)
	}
}

func TestServiceStoresResumeAndFallbackMetadata(t *testing.T) {
	ctx := context.Background()
	runtime := cruxapi.DefaultRuntimeConfig()
	st, err := store.NewFileStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(store.NewDomainStore(st), runner.NewCommandRunner(), testDiscoverer(), runtime, nil)
	_, err = svc.UpsertAgent(ctx, cruxapi.Agent{
		Name: "echo",
		Command: cruxapi.CommandSpec{
			Path: "/usr/bin/printf",
			Args: []string{"%s", "{prompt}"},
		},
		Status: cruxapi.AgentReady,
	})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := svc.SubmitExecution(ctx, cruxapi.SubmitExecutionRequest{
		AgentName:      "echo",
		Prompt:         "ok",
		ResumeSession:  "session-1",
		SourceExecID:   "exec_source",
		FallbackAgents: []string{"gemini", "gemini", "codex"},
		Wait:           false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if execution.ResumeSession != "session-1" || execution.SourceExecID != "exec_source" {
		t.Fatalf("unexpected execution metadata: %+v", execution)
	}
	if len(execution.FallbackAgents) != 2 || execution.FallbackAgents[0] != "gemini" || execution.FallbackAgents[1] != "codex" {
		t.Fatalf("unexpected fallback agents: %#v", execution.FallbackAgents)
	}
}

func TestServicePlansAndRecordsTTYExec(t *testing.T) {
	ctx := context.Background()
	runtime := cruxapi.DefaultRuntimeConfig()
	st, err := store.NewFileStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(store.NewDomainStore(st), runner.NewCommandRunner(), testDiscoverer(), runtime, nil)
	workDir := t.TempDir()
	_, err = svc.UpsertAgent(ctx, cruxapi.Agent{
		Name: "codex",
		Command: cruxapi.CommandSpec{
			Path: "/usr/bin/codex",
			Args: []string{"exec", "{prompt}"},
		},
		Status: cruxapi.AgentReady,
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := svc.AgentExecPlan(ctx, "codex", cruxapi.AgentExecPlanRequest{
		WorkingDir:    workDir,
		ResumeSession: "last",
		Args:          []string{"--no-alt-screen"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Command.WorkingDir != workDir {
		t.Fatalf("expected working dir %q, got %q", workDir, plan.Command.WorkingDir)
	}
	wantArgs := []string{"resume", "--all", "--last", "--no-alt-screen"}
	if len(plan.Command.Args) != len(wantArgs) {
		t.Fatalf("expected args %#v, got %#v", wantArgs, plan.Command.Args)
	}
	for i := range wantArgs {
		if plan.Command.Args[i] != wantArgs[i] {
			t.Fatalf("expected args %#v, got %#v", wantArgs, plan.Command.Args)
		}
	}
	record, err := svc.RecordAgentExec(ctx, "codex", cruxapi.AgentExecRecordRequest{
		WorkingDir:     workDir,
		ResumeSession:  "last",
		Args:           plan.Command.Args,
		Driver:         "script",
		TranscriptPath: "/tmp/codex.log",
		Transcript:     "interactive output",
		ExitCode:       0,
		StartedAt:      cruxapi.Now().Add(-time.Second),
		CompletedAt:    cruxapi.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.Execution.Status != cruxapi.ExecutionSucceeded {
		t.Fatalf("expected succeeded, got %s", record.Execution.Status)
	}
	if record.Execution.Stdout != "interactive output" {
		t.Fatalf("expected transcript in stdout, got %q", record.Execution.Stdout)
	}
	if record.Usage.ExecutionsTotal != 1 || record.Usage.Succeeded != 1 {
		t.Fatalf("unexpected usage after record: %+v", record.Usage)
	}
	events, err := svc.ListEvents(ctx, record.Execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestServiceRecoversInterruptedExecutions(t *testing.T) {
	ctx := context.Background()
	runtime := cruxapi.DefaultRuntimeConfig()
	st, err := store.NewFileStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	ds := store.NewDomainStore(st)
	queuedAt := cruxapi.Now().Add(-time.Minute)
	startedAt := queuedAt.Add(time.Second)
	running := cruxapi.Execution{
		ID:            "exec_running",
		AgentName:     "gemini",
		Prompt:        "hi",
		Status:        cruxapi.ExecutionRunning,
		QueuedAt:      queuedAt,
		StartedAt:     &startedAt,
		UpdatedAt:     startedAt,
		RuntimeConfig: runtime,
	}
	if err := ds.CreateExecution(ctx, running); err != nil {
		t.Fatal(err)
	}

	svc := New(ds, runner.NewCommandRunner(), testDiscoverer(), runtime, nil)
	if err := svc.RecoverInterruptedExecutions(ctx); err != nil {
		t.Fatal(err)
	}

	recovered, err := svc.GetExecution(ctx, running.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Status != cruxapi.ExecutionFailed {
		t.Fatalf("expected failed, got %s", recovered.Status)
	}
	if recovered.ExitCode != 124 {
		t.Fatalf("expected exit code 124, got %d", recovered.ExitCode)
	}
	if recovered.CompletedAt == nil {
		t.Fatal("expected completedAt to be set")
	}
	if recovered.Error != "execution interrupted by daemon restart" {
		t.Fatalf("unexpected error: %q", recovered.Error)
	}
	events, err := svc.ListEvents(ctx, running.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Type != cruxapi.EventExecutionFail {
		t.Fatalf("expected one failure event, got %+v", events)
	}
}
