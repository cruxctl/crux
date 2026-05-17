package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cruxctl/cruxd/pkg/cruxapi"
)

func TestDiscoverHelpDoesNotCallDaemon(t *testing.T) {
	var out, errOut bytes.Buffer
	code := New(&out, &errOut).Run(context.Background(), []string{"discover", "--help"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "crux discover") {
		t.Fatalf("expected discover usage, got %q", out.String())
	}
	if strings.Contains(errOut.String(), "connection refused") {
		t.Fatalf("help should not call daemon; stderr=%q", errOut.String())
	}
}

func TestNestedHelpDoesNotCallDaemon(t *testing.T) {
	var out, errOut bytes.Buffer
	code := New(&out, &errOut).Run(context.Background(), []string{"agents", "describe", "--help"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "crux agents describe <name>") {
		t.Fatalf("expected agents describe usage, got %q", out.String())
	}
}

func TestAgentGroupWithoutNamePrintsHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	code := New(&out, &errOut).Run(context.Background(), []string{"agent"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "crux agent <name> usage") {
		t.Fatalf("expected agent usage help, got %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("expected no stderr, got %q", errOut.String())
	}
}

func TestAgentsHelpMentionsAgentUsage(t *testing.T) {
	var out, errOut bytes.Buffer
	code := New(&out, &errOut).Run(context.Background(), []string{"agents", "--help"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "crux agent <name> usage") {
		t.Fatalf("expected agents help to mention usage, got %q", out.String())
	}
}

func TestRunSendsCurrentWorkingDirectory(t *testing.T) {
	var out, errOut bytes.Buffer
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/agents" {
			if err := json.NewEncoder(w).Encode([]cruxapi.Agent{{Name: "codex", Status: cruxapi.AgentReady}}); err != nil {
				t.Fatal(err)
			}
			return
		}
		switch r.URL.Path {
		case "/v1/agents/codex/exec/plan":
			var req cruxapi.AgentExecPlanRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.WorkingDir != cwd {
				t.Fatalf("expected workingDir %q, got %q", cwd, req.WorkingDir)
			}
			if req.Prompt != "hi" {
				t.Fatalf("expected prompt hi, got %q", req.Prompt)
			}
			plan := cruxapi.AgentExecPlan{
				AgentName: "codex",
				Provider:  "codex",
				Command: cruxapi.CommandSpec{
					Path:       "/usr/bin/printf",
					Args:       []string{"hi"},
					WorkingDir: req.WorkingDir,
				},
				Prompt:    req.Prompt,
				Operation: req.Operation,
			}
			if err := json.NewEncoder(w).Encode(plan); err != nil {
				t.Fatal(err)
			}
		case "/v1/agents/codex/exec/record":
			var req cruxapi.AgentExecRecordRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.WorkingDir != cwd {
				t.Fatalf("expected record workingDir %q, got %q", cwd, req.WorkingDir)
			}
			execution := cruxapi.Execution{
				ID:            "exec_test",
				AgentName:     "codex",
				Prompt:        req.Prompt,
				WorkingDir:    req.WorkingDir,
				Status:        cruxapi.ExecutionSucceeded,
				QueuedAt:      cruxapi.Now(),
				UpdatedAt:     cruxapi.Now(),
				RuntimeConfig: cruxapi.DefaultRuntimeConfig(),
			}
			record := cruxapi.AgentExecRecordResponse{
				Execution: execution,
				Usage:     cruxapi.AgentUsage{AgentName: "codex", ExecutionsTotal: 1, Succeeded: 1},
				Cost:      cruxapi.AgentCostSnapshot{AgentName: "codex"},
			}
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(record); err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	code := New(&out, &errOut).Run(context.Background(), []string{"--server", server.URL, "run", "codex", "hi"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
}

func TestAgentExecDryRunUsesDaemonPlan(t *testing.T) {
	var out, errOut bytes.Buffer
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	planCalled := false
	recordCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/agents/codex/exec/plan":
			planCalled = true
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			var req cruxapi.AgentExecPlanRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatal(err)
			}
			if req.WorkingDir != cwd {
				t.Fatalf("expected workingDir %q, got %q", cwd, req.WorkingDir)
			}
			if req.ResumeSession != "last" {
				t.Fatalf("expected resume last, got %q", req.ResumeSession)
			}
			if strings.Join(req.Args, " ") != "--no-alt-screen" {
				t.Fatalf("unexpected provider args: %#v", req.Args)
			}
			plan := cruxapi.AgentExecPlan{
				AgentName: "codex",
				Provider:  "openai-codex",
				Command: cruxapi.CommandSpec{
					Path:       "/usr/bin/codex",
					Args:       []string{"resume", "--all", "--last", "--no-alt-screen"},
					WorkingDir: cwd,
				},
			}
			if err := json.NewEncoder(w).Encode(plan); err != nil {
				t.Fatal(err)
			}
		case "/v1/agents/codex/exec/record":
			recordCalled = true
			t.Fatalf("dry run should not record")
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	code := New(&out, &errOut).Run(context.Background(), []string{"--server", server.URL, "agent", "codex", "exec", "--dry-run", "--resume", "last", "--", "--no-alt-screen"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	if !planCalled || recordCalled {
		t.Fatalf("planCalled=%v recordCalled=%v", planCalled, recordCalled)
	}
	if !strings.Contains(out.String(), "Command: /usr/bin/codex resume --all --last --no-alt-screen") {
		t.Fatalf("expected dry-run command, got %q", out.String())
	}
}

func TestDiscoverRejectsUnexpectedArgsBeforeClient(t *testing.T) {
	var out, errOut bytes.Buffer
	code := New(&out, &errOut).Run(context.Background(), []string{"discover", "extra"})

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "usage: crux discover") {
		t.Fatalf("expected usage error, got %q", errOut.String())
	}
	if out.Len() != 0 {
		t.Fatalf("unexpected stdout: %q", out.String())
	}
}

func TestOutputFlagCanFollowCommand(t *testing.T) {
	opts, cmd, rest, err := parseRoot([]string{"agent", "gemini", "usage", "-o", "yaml"})
	if err != nil {
		t.Fatal(err)
	}
	rest, err = applyOutputFlag(opts.output, rest, &opts.output)
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "agent" {
		t.Fatalf("expected command agent, got %q", cmd)
	}
	if opts.output != "yaml" {
		t.Fatalf("expected yaml output, got %q", opts.output)
	}
	if strings.Join(rest, " ") != "gemini usage" {
		t.Fatalf("unexpected rest: %#v", rest)
	}
}

func TestOutputFlagCanAppearBeforeCommandArgs(t *testing.T) {
	opts, cmd, rest, err := parseRoot([]string{"run", "-o", "json", "gemini", "hi"})
	if err != nil {
		t.Fatal(err)
	}
	rest, err = applyOutputFlag(opts.output, rest, &opts.output)
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "run" {
		t.Fatalf("expected command run, got %q", cmd)
	}
	if opts.output != "json" {
		t.Fatalf("expected json output, got %q", opts.output)
	}
	if strings.Join(rest, " ") != "gemini hi" {
		t.Fatalf("unexpected rest: %#v", rest)
	}
}

func TestParseRunArgsSupportsResumeHistoryAndFallback(t *testing.T) {
	got, err := parseRunArgs([]string{"codex", "--from", "exec_1", "--prompt", "revise", "--resume", "last", "--fallback", "gemini,claude"})
	if err != nil {
		t.Fatal(err)
	}
	if got.AgentName != "codex" || got.SourceExecID != "exec_1" || got.Prompt != "revise" || got.ResumeSession != "last" {
		t.Fatalf("unexpected run options: %+v", got)
	}
	if strings.Join(got.FallbackAgents, ",") != "gemini,claude" {
		t.Fatalf("unexpected fallbacks: %#v", got.FallbackAgents)
	}
}

func TestParseAgentExecArgsAllowsFlagsAfterPrompt(t *testing.T) {
	got, err := parseAgentExecArgs([]string{"insert into session", "--resume", "last", "--dry-run", "--", "--no-alt-screen"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Prompt != "insert into session" || got.ResumeSession != "last" || !got.DryRun {
		t.Fatalf("unexpected exec options: %+v", got)
	}
	if strings.Join(got.ProviderArgs, " ") != "--no-alt-screen" {
		t.Fatalf("unexpected provider args: %#v", got.ProviderArgs)
	}
}

func TestDirectTTYWritesTranscript(t *testing.T) {
	var out, errOut bytes.Buffer
	transcript := filepath.Join(t.TempDir(), "tty.log")
	result := New(&out, &errOut).runDirectTTY(cruxapi.CommandSpec{
		Path: "/usr/bin/printf",
		Args: []string{"hello"},
	}, agentExecOptions{TranscriptPath: transcript}, "direct")

	if result.ExitCode != 0 || result.Error != "" {
		t.Fatalf("unexpected direct result: %+v stderr=%q", result, errOut.String())
	}
	if out.String() != "hello" {
		t.Fatalf("expected stdout hello, got %q", out.String())
	}
	data, err := os.ReadFile(transcript)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected transcript hello, got %q", string(data))
	}
}

func TestFilterExecutions(t *testing.T) {
	executions := []cruxapi.Execution{
		{ID: "1", AgentName: "gemini", Status: cruxapi.ExecutionSucceeded},
		{ID: "2", AgentName: "codex", Status: cruxapi.ExecutionFailed},
		{ID: "3", AgentName: "codex", Status: cruxapi.ExecutionSucceeded},
	}
	got := filterExecutions(executions, psFilter{Agent: "codex", Last: 1})
	if len(got) != 1 || got[0].ID != "2" {
		t.Fatalf("unexpected filtered executions: %+v", got)
	}
	got = filterExecutions(executions, psFilter{Status: "succeeded"})
	if len(got) != 2 {
		t.Fatalf("expected two succeeded executions, got %+v", got)
	}
}
