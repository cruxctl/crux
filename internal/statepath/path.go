// Package statepath returns the canonical paths used by cruxd for state,
// config, logs, and per-entity directories rooted at $CRUX_HOME (default
// ~/.crux/). Hash-named subdirectories follow the Docker-volume mental
// model — see hash.go for the ID function.
package statepath

import (
	"os"
	"path/filepath"
)

// Root returns $CRUX_HOME if set, else $HOME/.crux.
func Root() string {
	if v := os.Getenv("CRUX_HOME"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".crux"
	}
	return filepath.Join(home, ".crux")
}

// ConfigPath is the daemon's YAML config.
func ConfigPath() string { return filepath.Join(Root(), "config.yaml") }

// LogFile returns ~/.crux/logs/<component>.log
func LogFile(component string) string {
	return filepath.Join(Root(), "logs", component+".log")
}

// PIDFile returns the cruxd PID file path.
func PIDFile() string { return filepath.Join(Root(), "daemon.pid") }

// DaemonSocket returns the Unix socket path for the daemon API.
func DaemonSocket() string { return filepath.Join(Root(), "daemon.sock") }

// GatewaySocket returns the Unix socket path for Crux Gateway.
func GatewaySocket() string { return filepath.Join(Root(), "gateway.sock") }

// StateRoot is ~/.crux/state — parent of all hash-named entity dirs.
func StateRoot() string { return filepath.Join(Root(), "state") }

// AgentDir returns the hash dir for a registered agent.
func AgentDir(sha string) string { return filepath.Join(StateRoot(), "agents", sha) }

// SessionDir returns the hash dir for a session.
func SessionDir(sha string) string { return filepath.Join(StateRoot(), "sessions", sha) }

// ProjectDir returns the hash dir for a project.
func ProjectDir(sha string) string { return filepath.Join(StateRoot(), "projects", sha) }

// MCPServerDir returns the hash dir for an MCP server record.
func MCPServerDir(sha string) string {
	return filepath.Join(StateRoot(), "gateway", "mcp", "servers", sha)
}

// AOSEventsDir is ~/.crux/state/aos/events.
func AOSEventsDir() string { return filepath.Join(StateRoot(), "aos", "events") }

// AgBOMDir returns the agbom dir for one of {agents, projects, sessions}.
func AgBOMDir(kind string) string {
	return filepath.Join(StateRoot(), "aos", "agbom", kind)
}

// PolicyDir is ~/.crux/state/policies.
func PolicyDir() string { return filepath.Join(StateRoot(), "policies") }

// IndexesDir is ~/.crux/state/indexes.
func IndexesDir() string { return filepath.Join(StateRoot(), "indexes") }

// EnsureDir creates the directory (and parents) with mode 0700.
func EnsureDir(p string) error { return os.MkdirAll(p, 0o700) }
