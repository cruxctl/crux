package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

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
	if runtime.GOOS == "windows" {
		if value := strings.TrimSpace(os.Getenv("APPDATA")); value != "" {
			return value
		}
	}
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, "Library", "Application Support")
		}
	}
	if value := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config")
}
