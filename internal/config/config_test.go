package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDaemonConfigReadsYAMLAndEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cruxd.yaml")
	state := filepath.Join(dir, "state.json")
	data := []byte(`
server:
  address: 127.0.0.1
  port: 7711
  readTimeoutSeconds: 5
  writeTimeoutSeconds: 60
  shutdownGraceSeconds: 3
store:
  path: ` + state + `
runtime:
  workerConcurrency: 2
  jobTimeoutSeconds: 10
  maxOutputBytes: 2048
  discoveryTimeoutSeconds: 2
  logLevel: info
  defaultNamespace: default
  allowShellCommands: false
  traceRetentionEntries: 100
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CRUX_SERVER_PORT", "7799")
	cfg, err := LoadDaemonConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 7799 {
		t.Fatalf("expected env override port 7799, got %d", cfg.Server.Port)
	}
	if cfg.Store.Path != state {
		t.Fatalf("expected state path %s, got %s", state, cfg.Store.Path)
	}
}

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
