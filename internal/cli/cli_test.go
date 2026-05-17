package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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
		if r.URL.Path != "/v1/executions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var req cruxapi.SubmitExecutionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.WorkingDir != cwd {
			t.Fatalf("expected workingDir %q, got %q", cwd, req.WorkingDir)
		}
		execution := cruxapi.Execution{
			ID:            "exec_test",
			AgentName:     req.AgentName,
			Prompt:        req.Prompt,
			WorkingDir:    req.WorkingDir,
			Status:        cruxapi.ExecutionSucceeded,
			QueuedAt:      cruxapi.Now(),
			UpdatedAt:     cruxapi.Now(),
			RuntimeConfig: cruxapi.DefaultRuntimeConfig(),
		}
		if err := json.NewEncoder(w).Encode(execution); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	code := New(&out, &errOut).Run(context.Background(), []string{"--server", server.URL, "run", "codex", "hi"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stdout=%q stderr=%q", code, out.String(), errOut.String())
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
