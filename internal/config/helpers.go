package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ExpandPath resolves ~ and relative paths to absolute paths.
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

// ResolveOptionalPath returns the explicit path if given, otherwise the
// defaultPath if it exists on disk.
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

func defaultDataHome() string {
	if runtime.GOOS == "windows" {
		if value := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); value != "" {
			return value
		}
	}
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, "Library", "Application Support")
		}
	}
	if value := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".local", "share")
}
