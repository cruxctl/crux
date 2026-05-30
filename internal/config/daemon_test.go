package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultsMatchBlueprint(t *testing.T) {
	t.Setenv("CRUX_HOME", t.TempDir())
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load default: %v", err)
	}
	if cfg.Daemon.Port != 4357 {
		t.Errorf("Daemon.Port = %d, want 4357", cfg.Daemon.Port)
	}
	if cfg.Console.Port != 4358 {
		t.Errorf("Console.Port = %d, want 4358", cfg.Console.Port)
	}
	if cfg.Gateway.Port != 4360 {
		t.Errorf("Gateway.Port = %d, want 4360", cfg.Gateway.Port)
	}
	if cfg.State.RetentionDays != 90 {
		t.Errorf("State.RetentionDays = %d, want 90", cfg.State.RetentionDays)
	}
	if cfg.State.TranscriptMode != "full" {
		t.Errorf("State.TranscriptMode = %q, want full", cfg.State.TranscriptMode)
	}
	if cfg.API.AuthMode != "none" {
		t.Errorf("API.AuthMode = %q, want none", cfg.API.AuthMode)
	}
}

func TestLoad_FromYAMLOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	body := `version: "0.1"
daemon:
  port: 9999
state:
  retention_days: 7
gateway:
  guardian_mode: enforcing
api:
  api_keys: ["test-key-1"]
`
	if err := os.WriteFile(cfgFile, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Daemon.Port != 9999 {
		t.Errorf("Daemon.Port = %d, want 9999", cfg.Daemon.Port)
	}
	if cfg.State.RetentionDays != 7 {
		t.Errorf("State.RetentionDays = %d, want 7", cfg.State.RetentionDays)
	}
	if cfg.Gateway.GuardianMode != "enforcing" {
		t.Errorf("Gateway.GuardianMode = %q, want enforcing", cfg.Gateway.GuardianMode)
	}
	if len(cfg.API.APIKeys) != 1 || cfg.API.APIKeys[0] != "test-key-1" {
		t.Errorf("API.APIKeys = %v, want [test-key-1]", cfg.API.APIKeys)
	}
}

func TestLoad_EnvOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte("daemon:\n  port: 4357\n"), 0o600)
	t.Setenv("CRUXD_DAEMON_PORT", "5555")
	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Daemon.Port != 5555 {
		t.Errorf("Daemon.Port = %d, want 5555 (env override)", cfg.Daemon.Port)
	}
}

func TestValidate_RejectsBadGuardianMode(t *testing.T) {
	cfg := Defaults()
	cfg.Gateway.GuardianMode = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Errorf("Validate should reject bogus guardian_mode")
	}
}
