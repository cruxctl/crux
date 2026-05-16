package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cruxctl/crux/internal/domain"
	"gopkg.in/yaml.v3"
)

type DaemonConfig struct {
	Server   ServerConfig         `json:"server" yaml:"server"`
	Store    StoreConfig          `json:"store" yaml:"store"`
	Runtime  domain.RuntimeConfig `json:"runtime" yaml:"runtime"`
	Security SecurityConfig       `json:"security" yaml:"security"`
}

type ServerConfig struct {
	Address                string `json:"address" yaml:"address"`
	Port                   int    `json:"port" yaml:"port"`
	ReadTimeoutSeconds     int    `json:"readTimeoutSeconds" yaml:"readTimeoutSeconds"`
	WriteTimeoutSeconds    int    `json:"writeTimeoutSeconds" yaml:"writeTimeoutSeconds"`
	ShutdownGraceSeconds   int    `json:"shutdownGraceSeconds" yaml:"shutdownGraceSeconds"`
	RuntimeConfigReloadURL string `json:"runtimeConfigReloadUrl,omitempty" yaml:"runtimeConfigReloadUrl,omitempty"`
}

type StoreConfig struct {
	Path string `json:"path" yaml:"path"`
}

type SecurityConfig struct {
	APIKey string `json:"apiKey" yaml:"apiKey"`
}

func DefaultDaemonConfig() DaemonConfig {
	return DaemonConfig{
		Server: ServerConfig{
			Address:              "127.0.0.1",
			Port:                 7700,
			ReadTimeoutSeconds:   15,
			WriteTimeoutSeconds:  300,
			ShutdownGraceSeconds: 15,
		},
		Store: StoreConfig{
			Path: filepath.Join(defaultDataHome(), "crux", "state.json"),
		},
		Runtime: domain.DefaultRuntimeConfig(),
	}
}

func LoadDaemonConfig(path string) (DaemonConfig, error) {
	cfg := DefaultDaemonConfig()
	resolved, err := ResolveOptionalPath(path, filepath.Join(defaultConfigHome(), "crux", "cruxd.yaml"))
	if err != nil {
		return cfg, err
	}
	if resolved != "" {
		data, err := os.ReadFile(resolved)
		if err != nil {
			return cfg, fmt.Errorf("read daemon config %s: %w", resolved, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse daemon config %s: %w", resolved, err)
		}
	}
	applyDaemonEnv(&cfg)
	cfg.Store.Path, err = ExpandPath(cfg.Store.Path)
	if err != nil {
		return cfg, err
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c DaemonConfig) ListenAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Address, c.Server.Port)
}

func (c DaemonConfig) Validate() error {
	if strings.TrimSpace(c.Server.Address) == "" {
		return errors.New("server.address is required")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.New("server.port must be between 1 and 65535")
	}
	if c.Server.ReadTimeoutSeconds < 1 {
		return errors.New("server.readTimeoutSeconds must be >= 1")
	}
	if c.Server.WriteTimeoutSeconds < 1 {
		return errors.New("server.writeTimeoutSeconds must be >= 1")
	}
	if c.Server.ShutdownGraceSeconds < 1 {
		return errors.New("server.shutdownGraceSeconds must be >= 1")
	}
	if strings.TrimSpace(c.Store.Path) == "" {
		return errors.New("store.path is required")
	}
	return c.Runtime.Validate()
}

func applyDaemonEnv(cfg *DaemonConfig) {
	setString("CRUX_SERVER_ADDRESS", &cfg.Server.Address)
	setInt("CRUX_SERVER_PORT", &cfg.Server.Port)
	setInt("CRUX_SERVER_READ_TIMEOUT_SECONDS", &cfg.Server.ReadTimeoutSeconds)
	setInt("CRUX_SERVER_WRITE_TIMEOUT_SECONDS", &cfg.Server.WriteTimeoutSeconds)
	setInt("CRUX_SERVER_SHUTDOWN_GRACE_SECONDS", &cfg.Server.ShutdownGraceSeconds)
	setString("CRUX_STORE_PATH", &cfg.Store.Path)
	setString("CRUX_API_KEY", &cfg.Security.APIKey)
	setInt("CRUX_WORKER_CONCURRENCY", &cfg.Runtime.WorkerConcurrency)
	setInt("CRUX_JOB_TIMEOUT_SECONDS", &cfg.Runtime.JobTimeoutSeconds)
	setInt("CRUX_MAX_OUTPUT_BYTES", &cfg.Runtime.MaxOutputBytes)
	setInt("CRUX_DISCOVERY_TIMEOUT_SECONDS", &cfg.Runtime.DiscoveryTimeoutSecs)
	setString("CRUX_LOG_LEVEL", &cfg.Runtime.LogLevel)
	setString("CRUX_DEFAULT_NAMESPACE", &cfg.Runtime.DefaultNamespace)
	setBool("CRUX_ALLOW_SHELL_COMMANDS", &cfg.Runtime.AllowShellCommands)
	setInt("CRUX_TRACE_RETENTION_ENTRIES", &cfg.Runtime.TraceRetentionEntries)
}

func setString(key string, target *string) {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		*target = value
	}
}

func setInt(key string, target *int) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return
	}
	*target = parsed
}

func setBool(key string, target *bool) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return
	}
	*target = parsed
}

func ResolveOptionalPath(path, defaultPath string) (string, error) {
	if strings.TrimSpace(path) != "" {
		return ExpandPath(path)
	}
	resolved, err := ExpandPath(defaultPath)
	if err != nil {
		return "", err
	}
	if _, statErr := os.Stat(resolved); statErr == nil {
		return resolved, nil
	} else if errors.Is(statErr, os.ErrNotExist) {
		return "", nil
	} else {
		return "", fmt.Errorf("stat config %s: %w", resolved, statErr)
	}
}

func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return filepath.Abs(path)
}

func defaultConfigHome() string {
	if value := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config")
}

func defaultDataHome() string {
	if value := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".local", "share")
}
