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

func New(opts Options, stderr io.Writer) (*slog.Logger, func() error, error) {
	level, err := parseLevel(opts.Level)
	if err != nil {
		return nil, nil, err
	}
	if opts.MaxSizeMB <= 0 {
		opts.MaxSizeMB = 10
	}
	if opts.MaxBackups < 0 {
		opts.MaxBackups = 0
	}
	if strings.TrimSpace(opts.Format) == "" {
		opts.Format = "text"
	}

	writers := []io.Writer{stderr}
	var closer func() error = func() error { return nil }
	if strings.TrimSpace(opts.File) != "" {
		rotator, err := NewRotatingFile(opts.File, int64(opts.MaxSizeMB)*1024*1024, opts.MaxBackups)
		if err != nil {
			return nil, nil, err
		}
		writers = append(writers, rotator)
		closer = rotator.Close
	}
	handlerOptions := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	writer := io.MultiWriter(writers...)
	switch strings.ToLower(strings.TrimSpace(opts.Format)) {
	case "json":
		handler = slog.NewJSONHandler(writer, handlerOptions)
	case "text":
		handler = slog.NewTextHandler(writer, handlerOptions)
	default:
		return nil, nil, fmt.Errorf("unsupported log format %q", opts.Format)
	}
	return slog.New(handler), closer, nil
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
		return nil, fmt.Errorf("stat log file %s: %w", path, err)
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
					return fmt.Errorf("rotate log backup: %w", err)
				}
			}
		}
		if _, err := os.Stat(w.path); err == nil {
			if err := os.Rename(w.path, backupPath(w.path, 1)); err != nil {
				return fmt.Errorf("rotate log file: %w", err)
			}
		}
	} else {
		_ = os.Remove(w.path)
	}
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open rotated log file: %w", err)
	}
	w.file = file
	w.size = 0
	return nil
}

func backupPath(path string, index int) string {
	return fmt.Sprintf("%s.%d", path, index)
}
