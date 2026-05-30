package envpath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLookupFindsNVMStyleBinaryOutsidePATH(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, ".nvm", "versions", "node", "v99.0.0", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(binDir, "codex")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", filepath.Join(home, "empty"))

	got, err := Lookup("codex")
	if err != nil {
		t.Fatalf("expected codex lookup to succeed: %v", err)
	}
	if got != binaryPath {
		t.Fatalf("expected %q, got %q", binaryPath, got)
	}
}

func TestCommandEnvPrependsCommandDirectoryToPATH(t *testing.T) {
	commandDir := filepath.Join(t.TempDir(), "bin")
	env := CommandEnv([]string{"PATH=/usr/bin", "FOO=bar"}, filepath.Join(commandDir, "gemini"), map[string]string{"BAZ": "qux"})

	pathValue := ""
	for _, item := range env {
		if strings.HasPrefix(item, "PATH=") {
			pathValue = strings.TrimPrefix(item, "PATH=")
			break
		}
	}
	if pathValue == "" {
		t.Fatal("expected PATH in command environment")
	}
	if got := filepath.SplitList(pathValue)[0]; got != commandDir {
		t.Fatalf("expected command dir first in PATH, got %q", got)
	}
	if !strings.Contains(pathValue, "/usr/bin") {
		t.Fatalf("expected original PATH to be preserved, got %q", pathValue)
	}
}
