package config

import (
	"path/filepath"
	"testing"
)

func TestCLIConfigDefaultContext(t *testing.T) {
	cfg, _, err := LoadCLIConfig(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, name, err := cfg.ActiveContext("")
	if err != nil {
		t.Fatal(err)
	}
	if name != "local" {
		t.Fatalf("expected local context, got %s", name)
	}
	if ctx.ServerURL != "http://127.0.0.1:7700" {
		t.Fatalf("unexpected server URL %s", ctx.ServerURL)
	}
}
