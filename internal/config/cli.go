package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type CLIConfig struct {
	CurrentContext string                `json:"currentContext" yaml:"currentContext"`
	Contexts       map[string]CLIContext `json:"contexts" yaml:"contexts"`
}

type CLIContext struct {
	ServerURL string `json:"serverUrl" yaml:"serverUrl"`
	APIKey    string `json:"apiKey,omitempty" yaml:"apiKey,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

func DefaultCLIConfig() CLIConfig {
	return CLIConfig{
		CurrentContext: "local",
		Contexts: map[string]CLIContext{
			"local": {
				ServerURL: "http://127.0.0.1:4357",
				Namespace: "default",
			},
		},
	}
}

func DefaultCLIConfigPath() string {
	return filepath.Join(defaultConfigHome(), "crux", "config.yaml")
}

func LoadCLIConfig(path string) (CLIConfig, string, error) {
	cfg := DefaultCLIConfig()
	resolved := path
	var err error
	if strings.TrimSpace(resolved) == "" {
		resolved = DefaultCLIConfigPath()
	}
	resolved, err = ExpandPath(resolved)
	if err != nil {
		return cfg, "", err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, resolved, nil
		}
		return cfg, resolved, fmt.Errorf("read cli config %s: %w", resolved, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, resolved, fmt.Errorf("parse cli config %s: %w", resolved, err)
	}
	if cfg.Contexts == nil {
		cfg.Contexts = map[string]CLIContext{}
	}
	return cfg, resolved, nil
}

func SaveCLIConfig(path string, cfg CLIConfig) error {
	if cfg.Contexts == nil {
		cfg.Contexts = map[string]CLIContext{}
	}
	resolved := path
	if strings.TrimSpace(resolved) == "" {
		resolved = DefaultCLIConfigPath()
	}
	var err error
	resolved, err = ExpandPath(resolved)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return fmt.Errorf("create cli config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal cli config: %w", err)
	}
	if err := os.WriteFile(resolved, data, 0o600); err != nil {
		return fmt.Errorf("write cli config %s: %w", resolved, err)
	}
	return nil
}

func (c CLIConfig) ActiveContext(name string) (CLIContext, string, error) {
	selected := strings.TrimSpace(name)
	if selected == "" {
		selected = c.CurrentContext
	}
	if selected == "" {
		selected = "local"
	}
	ctx, ok := c.Contexts[selected]
	if !ok {
		return CLIContext{}, selected, fmt.Errorf("context %q not found", selected)
	}
	if strings.TrimSpace(ctx.ServerURL) == "" {
		return CLIContext{}, selected, fmt.Errorf("context %q has no serverUrl", selected)
	}
	return ctx, selected, nil
}
