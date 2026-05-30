package mcp

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors agent data directories and rescans on change.
type Watcher struct {
	store    *JsonStore
	adapters []AgentAdapter
	logger   *slog.Logger
}

// NewWatcher creates a watcher.
func NewWatcher(store *JsonStore, adapters []AgentAdapter, logger *slog.Logger) *Watcher {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return &Watcher{store: store, adapters: adapters, logger: logger}
}

// Start begins watching and returns immediately.
func (w *Watcher) Start() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	go w.loop(watcher)
	for _, adapter := range w.adapters {
		for _, path := range adapter.DataPaths() {
			_ = watcher.Add(path)
			// Also add existing subdirectories.
			_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err == nil && info.IsDir() {
					_ = watcher.Add(p)
				}
				return nil
			})
		}
	}
	return nil
}

func (w *Watcher) loop(watcher *fsnotify.Watcher) {
	debounce := time.NewTimer(0)
	<-debounce.C
	pending := false
	for {
		select {
		case evt, ok := <-watcher.Events:
			if !ok {
				return
			}
			if evt.Op&fsnotify.Create == fsnotify.Create {
				if info, err := os.Stat(evt.Name); err == nil && info.IsDir() {
					_ = watcher.Add(evt.Name)
				}
			}
			if !pending {
				pending = true
				debounce = time.NewTimer(250 * time.Millisecond)
			}
		case <-debounce.C:
			if pending {
				pending = false
				if err := w.Rescan(); err != nil {
					w.logger.Error("rescan failed", "error", err)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("fsnotify error", "error", err)
		}
	}
}

// Rescan parses all adapter data paths and merges into the store.
func (w *Watcher) Rescan() error {
	for _, adapter := range w.adapters {
		for _, path := range adapter.DataPaths() {
			_ = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() || !adapter.Matches(p) {
					return nil
				}
				sessions, err := adapter.ParseFile(p)
				if err != nil {
					return nil
				}
				for _, sess := range sessions {
					_ = w.store.SaveSession(adapter.ID(), sess)
				}
				return nil
			})
		}
	}
	return nil
}
