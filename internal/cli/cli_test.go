package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestDiscoverHelpDoesNotRunDiscovery(t *testing.T) {
	var out, errOut bytes.Buffer
	code := New(&out, &errOut).Run(context.Background(), []string{"discover", "--help"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "crux discover") {
		t.Fatalf("expected discover usage, got %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("expected no stderr, got %q", errOut.String())
	}
}

func TestLegacyAgentsCommandRemoved(t *testing.T) {
	var out, errOut bytes.Buffer
	code := New(&out, &errOut).Run(context.Background(), []string{"agents", "--help"})

	if code == 0 {
		t.Fatalf("expected removed command to fail; stdout=%q", out.String())
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("expected unknown command error, got %q", errOut.String())
	}
}

func TestAgentShorthandHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	code := New(&out, &errOut).Run(context.Background(), []string{"claude-code", "--help"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "crux claude-code usage") {
		t.Fatalf("expected shorthand usage, got %q", out.String())
	}
}

func TestAgentNestedHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	code := New(&out, &errOut).Run(context.Background(), []string{"gemini-cli", "conversations", "--help"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "crux gemini-cli conversations ls") {
		t.Fatalf("expected conversations usage, got %q", out.String())
	}
}

func TestOutputFlagCanFollowAgentCommand(t *testing.T) {
	opts, cmd, rest, err := parseRoot([]string{"gemini-cli", "usage", "-o", "yaml"})
	if err != nil {
		t.Fatal(err)
	}
	rest, err = applyOutputFlag(opts.output, rest, &opts.output)
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "gemini-cli" {
		t.Fatalf("expected agent command, got %q", cmd)
	}
	if opts.output != "yaml" {
		t.Fatalf("expected yaml output, got %q", opts.output)
	}
	if strings.Join(rest, " ") != "usage" {
		t.Fatalf("unexpected rest: %#v", rest)
	}
}

func TestOutputFlagCanAppearBeforeAgentArgs(t *testing.T) {
	opts, cmd, rest, err := parseRoot([]string{"codex", "-o", "json", "describe"})
	if err != nil {
		t.Fatal(err)
	}
	rest, err = applyOutputFlag(opts.output, rest, &opts.output)
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "codex" {
		t.Fatalf("expected command codex, got %q", cmd)
	}
	if opts.output != "json" {
		t.Fatalf("expected json output, got %q", opts.output)
	}
	if strings.Join(rest, " ") != "describe" {
		t.Fatalf("unexpected rest: %#v", rest)
	}
}

func TestParsePTYExecArgs(t *testing.T) {
	got, err := parsePTYExecArgs([]string{"--repo", ".", "--timeout", "30", "--", "--no-alt-screen"})
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkDir == "" {
		t.Fatalf("expected absolute workdir")
	}
	if got.Timeout != 30*time.Second {
		t.Fatalf("unexpected timeout: %s", got.Timeout)
	}
	if strings.Join(got.ProviderArgs, " ") != "--no-alt-screen" {
		t.Fatalf("unexpected provider args: %#v", got.ProviderArgs)
	}
}
