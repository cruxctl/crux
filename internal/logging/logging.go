package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type Options struct {
	Level      string
	File       string
	Format     string
	MaxSizeMB  int
	MaxBackups int
}

func DefaultLogFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "crux.log"
	}
	switch runtime.GOOS {
	case "windows":
		if base := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); base != "" {
			return filepath.Join(base, "Crux", "logs", "crux.log")
		}
		return filepath.Join(home, "AppData", "Local", "Crux", "logs", "crux.log")
	case "darwin":
		return filepath.Join(home, "Library", "Logs", "Crux", "crux.log")
	default:
		if base := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); base != "" {
			return filepath.Join(base, "crux", "crux.log")
		}
		return filepath.Join(home, ".local", "state", "crux", "crux.log")
	}
}

func New(opts Options) (*slog.Logger, func() error, error) {
	level, err := parseLevel(firstNonEmpty(opts.Level, envOrDefault("CRUX_LOG_LEVEL", "info")))
	if err != nil {
		return nil, nil, err
	}
	file := firstNonEmpty(opts.File, envOrDefault("CRUX_LOG_FILE", DefaultLogFile()))
	format := firstNonEmpty(opts.Format, envOrDefault("CRUX_LOG_FORMAT", "text"))
	if opts.MaxSizeMB <= 0 {
		opts.MaxSizeMB = envInt("CRUX_LOG_MAX_SIZE_MB", 10)
	}
	if opts.MaxBackups <= 0 {
		opts.MaxBackups = envInt("CRUX_LOG_MAX_BACKUPS", 5)
	}
	writer, closeFn, err := logWriter(file, opts.MaxSizeMB, opts.MaxBackups)
	if err != nil {
		return nil, nil, err
	}
	handlerOptions := &slog.HandlerOptions{Level: level}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return slog.New(slog.NewJSONHandler(writer, handlerOptions)), closeFn, nil
	case "text":
		return slog.New(slog.NewTextHandler(writer, handlerOptions)), closeFn, nil
	default:
		_ = closeFn()
		return nil, nil, fmt.Errorf("unsupported log format %q", format)
	}
}

func logWriter(path string, maxSizeMB, maxBackups int) (io.Writer, func() error, error) {
	if strings.TrimSpace(path) == "" || strings.EqualFold(path, "none") {
		return io.Discard, func() error { return nil }, nil
	}
	rotator, err := NewRotatingFile(path, int64(maxSizeMB)*1024*1024, maxBackups)
	if err != nil {
		return nil, nil, err
	}
	return rotator, rotator.Close, nil
}

func parseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		if parsed, err := strconv.Atoi(value); err == nil {
			return slog.Level(parsed), nil
		}
		return slog.LevelInfo, fmt.Errorf("unsupported log level %q", value)
	}
}

type RotatingFile struct {
	path       string
	maxBytes   int64
	maxBackups int
	mu         sync.Mutex
	file       *os.File
	size       int64
}

func NewRotatingFile(path string, maxBytes int64, maxBackups int) (*RotatingFile, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("log max bytes must be > 0")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log file %s: %w", path, err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &RotatingFile{path: path, maxBytes: maxBytes, maxBackups: maxBackups, file: file, size: info.Size()}, nil
}

func (w *RotatingFile) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size+int64(len(p)) > w.maxBytes {
		if err := w.rotateLocked(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *RotatingFile) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *RotatingFile) rotateLocked() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
	}
	if w.maxBackups > 0 {
		_ = os.Remove(backupPath(w.path, w.maxBackups))
		for i := w.maxBackups - 1; i >= 1; i-- {
			oldPath := backupPath(w.path, i)
			if _, err := os.Stat(oldPath); err == nil {
				if err := os.Rename(oldPath, backupPath(w.path, i+1)); err != nil {
					return err
				}
			}
		}
		if _, err := os.Stat(w.path); err == nil {
			if err := os.Rename(w.path, backupPath(w.path, 1)); err != nil {
				return err
			}
		}
	} else {
		_ = os.Remove(w.path)
	}
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	w.file = file
	w.size = 0
	return nil
}

func backupPath(path string, index int) string {
	return fmt.Sprintf("%s.%d", path, index)
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
