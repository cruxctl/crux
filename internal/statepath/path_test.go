package statepath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoot_DefaultsToHomeDotCrux(t *testing.T) {
	t.Setenv("CRUX_HOME", "")
	t.Setenv("HOME", "/tmp/test-crux-home")
	got := Root()
	want := "/tmp/test-crux-home/.crux"
	if got != want {
		t.Errorf("Root() = %q, want %q", got, want)
	}
}

func TestRoot_HonorsCruxHomeEnv(t *testing.T) {
	t.Setenv("CRUX_HOME", "/var/lib/crux")
	got := Root()
	if got != "/var/lib/crux" {
		t.Errorf("Root() = %q, want /var/lib/crux", got)
	}
}

func TestConfigPath(t *testing.T) {
	t.Setenv("CRUX_HOME", "/tmp/x")
	got := ConfigPath()
	want := "/tmp/x/config.yaml"
	if got != want {
		t.Errorf("ConfigPath() = %q, want %q", got, want)
	}
}

func TestLogFile(t *testing.T) {
	t.Setenv("CRUX_HOME", "/tmp/x")
	got := LogFile("cruxd")
	want := "/tmp/x/logs/cruxd.log"
	if got != want {
		t.Errorf("LogFile() = %q, want %q", got, want)
	}
}

func TestAgentDir_UsesHashShort32(t *testing.T) {
	t.Setenv("CRUX_HOME", "/tmp/x")
	got := AgentDir("abc123")
	if filepath.Dir(got) != "/tmp/x/state/agents" {
		t.Errorf("AgentDir parent = %q, want /tmp/x/state/agents", filepath.Dir(got))
	}
	if filepath.Base(got) != "abc123" {
		t.Errorf("AgentDir leaf = %q, want abc123", filepath.Base(got))
	}
}

func TestEnsureDir_CreatesWithMode0700(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "c")
	if err := EnsureDir(target); err != nil {
		t.Fatalf("EnsureDir error: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("perm = %v, want 0700", perm)
	}
}
