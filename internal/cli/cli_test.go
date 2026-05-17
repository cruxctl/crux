package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
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
