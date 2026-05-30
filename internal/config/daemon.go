// Package config loads cruxd's daemon configuration from defaults, a YAML
// file, environment variables, and (later) CLI flags / runtime patches.
// Precedence highest-first: flags > env > runtime > YAML > defaults.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/cruxctl/crux/internal/statepath"
)

type DaemonConfig struct {
	Version    string            `yaml:"version"`
	Daemon     DaemonSection     `yaml:"daemon"`
	State      StateSection      `yaml:"state"`
	Console    ConsoleSection    `yaml:"console"`
	PTY        PTYSection        `yaml:"pty"`
	Gateway    GatewaySection    `yaml:"gateway"`
	AOS        AOSSection        `yaml:"aos"`
	Usage      UsageSection      `yaml:"usage"`
	Scheduler  SchedulerSection  `yaml:"scheduler"`
	Enterprise EnterpriseSection `yaml:"enterprise"`
	API        APISection        `yaml:"api"`
}

type DaemonSection struct {
	Host                 string `yaml:"host"`
	Port                 int    `yaml:"port"`
	Socket               string `yaml:"socket"`
	LogLevel             string `yaml:"log_level"`
	LogFormat            string `yaml:"log_format"`
	LogFile              string `yaml:"log_file"`
	ReadTimeoutSeconds   int    `yaml:"read_timeout_seconds,omitempty"`
	WriteTimeoutSeconds  int    `yaml:"write_timeout_seconds,omitempty"`
	ShutdownGraceSeconds int    `yaml:"shutdown_grace_seconds,omitempty"`
}

type StateSection struct {
	Root           string `yaml:"root"`
	RetentionDays  int    `yaml:"retention_days"`
	TranscriptMode string `yaml:"transcript_mode"` // full|redacted|metadata_only|disabled
	RedactSecrets  bool   `yaml:"redact_secrets"`
}

type ConsoleSection struct {
	Enabled     bool   `yaml:"enabled"`
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	UpstreamURL string `yaml:"upstream_url"`
}

type PTYSection struct {
	DefaultTimeoutSeconds int  `yaml:"default_timeout_seconds"`
	CaptureANSI           bool `yaml:"capture_ansi"`
	CaptureRaw            bool `yaml:"capture_raw"`
	MaxBufferMB           int  `yaml:"max_buffer_mb"`
	DefaultRows           int  `yaml:"default_rows"`
	DefaultCols           int  `yaml:"default_cols"`
}

type GatewaySection struct {
	Enabled      bool          `yaml:"enabled"`
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	Socket       string        `yaml:"socket"`
	GuardianMode string        `yaml:"guardian_mode"` // passive|enforcing|strict
	Inject       InjectSection `yaml:"inject"`
}

type InjectSection struct {
	EnabledByDefault bool     `yaml:"enabled_by_default"`
	Targets          []string `yaml:"targets"`
}

type AOSSection struct {
	Enabled      bool `yaml:"enabled"`
	EmitOTel     bool `yaml:"emit_otel"`
	EmitOCSF     bool `yaml:"emit_ocsf"`
	AgBOMEnabled bool `yaml:"agbom_enabled"`
}

type UsageSection struct {
	Enabled bool         `yaml:"enabled"`
	Budgets UsageBudgets `yaml:"budgets"`
}

type UsageBudgets struct {
	DailyUSDWarn  float64 `yaml:"daily_usd_warn"`
	DailyUSDBlock float64 `yaml:"daily_usd_block"`
}

type SchedulerSection struct {
	Enabled bool      `yaml:"enabled"`
	Jobs    []JobSpec `yaml:"jobs"`
}

type JobSpec struct {
	ID         string   `yaml:"id"`
	Every      string   `yaml:"every"`
	Collectors []string `yaml:"collectors"`
	Agents     []string `yaml:"agents"`
	Probes     []string `yaml:"probes"`
	Mode       string   `yaml:"mode"`
	Builtin    string   `yaml:"builtin"`
}

type EnterpriseSection struct {
	Enabled             bool   `yaml:"enabled"`
	Mode                string `yaml:"mode"`
	Endpoint            string `yaml:"endpoint"`
	OrgID               string `yaml:"org_id"`
	EnrollmentTokenPath string `yaml:"enrollment_token_path"`
}

type APISection struct {
	AuthMode    string   `yaml:"auth_mode"` // none|api_key|mtls|oidc
	APIKeys     []string `yaml:"api_keys"`
	CORSOrigins []string `yaml:"cors_origins"`
}

// Defaults returns a fully-populated config with blueprint defaults.
func Defaults() DaemonConfig {
	return DaemonConfig{
		Version: "0.1",
		Daemon: DaemonSection{
			Host:                 "127.0.0.1",
			Port:                 4357,
			Socket:               statepath.DaemonSocket(),
			LogLevel:             "info",
			LogFormat:            "json",
			LogFile:              statepath.LogFile("cruxd"),
			ReadTimeoutSeconds:   15,
			WriteTimeoutSeconds:  300,
			ShutdownGraceSeconds: 15,
		},
		State: StateSection{
			Root:           statepath.Root(),
			RetentionDays:  90,
			TranscriptMode: "full",
			RedactSecrets:  true,
		},
		Console: ConsoleSection{
			Enabled:     true,
			Host:        "127.0.0.1",
			Port:        4358,
			UpstreamURL: "http://127.0.0.1:4358",
		},
		PTY: PTYSection{
			DefaultTimeoutSeconds: 300,
			CaptureANSI:           true,
			CaptureRaw:            true,
			MaxBufferMB:           100,
			DefaultRows:           40,
			DefaultCols:           120,
		},
		Gateway: GatewaySection{
			Enabled:      true,
			Host:         "127.0.0.1",
			Port:         4360,
			Socket:       statepath.GatewaySocket(),
			GuardianMode: "passive",
			Inject:       InjectSection{EnabledByDefault: false, Targets: nil},
		},
		AOS: AOSSection{
			Enabled:      true,
			EmitOTel:     true,
			EmitOCSF:     true,
			AgBOMEnabled: true,
		},
		Usage: UsageSection{
			Enabled: true,
			Budgets: UsageBudgets{DailyUSDWarn: 25, DailyUSDBlock: 100},
		},
		Scheduler:  SchedulerSection{Enabled: true},
		Enterprise: EnterpriseSection{Enabled: false, Mode: "metadata_only"},
		API: APISection{
			AuthMode:    "none",
			CORSOrigins: []string{"http://127.0.0.1:4358"},
		},
	}
}

// Load reads YAML from path (empty → defaults only), applies env overrides,
// validates, and returns the config.
func Load(path string) (DaemonConfig, error) {
	cfg := Defaults()
	if path != "" {
		body, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return cfg, fmt.Errorf("read %s: %w", path, err)
		}
		if len(body) > 0 {
			if err := yaml.Unmarshal(body, &cfg); err != nil {
				return cfg, fmt.Errorf("parse %s: %w", path, err)
			}
		}
	}
	applyEnv(&cfg)
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func applyEnv(c *DaemonConfig) {
	if v := os.Getenv("CRUXD_DAEMON_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Daemon.Port = n
		}
	}
	if v := os.Getenv("CRUXD_LOG_LEVEL"); v != "" {
		c.Daemon.LogLevel = v
	}
	if v := os.Getenv("CRUXD_API_AUTH_MODE"); v != "" {
		c.API.AuthMode = v
	}
}

func (c DaemonConfig) ListenAddress() string {
	return fmt.Sprintf("%s:%d", c.Daemon.Host, c.Daemon.Port)
}

func (c DaemonConfig) Validate() error {
	switch c.Gateway.GuardianMode {
	case "passive", "enforcing", "strict":
	default:
		return errors.New("gateway.guardian_mode must be passive|enforcing|strict")
	}
	switch c.State.TranscriptMode {
	case "full", "redacted", "metadata_only", "disabled":
	default:
		return errors.New("state.transcript_mode must be full|redacted|metadata_only|disabled")
	}
	switch c.API.AuthMode {
	case "none", "api_key", "mtls", "oidc":
	default:
		return errors.New("api.auth_mode must be none|api_key|mtls|oidc")
	}
	if c.Daemon.Port <= 0 || c.Daemon.Port > 65535 {
		return errors.New("daemon.port out of range")
	}
	return nil
}
