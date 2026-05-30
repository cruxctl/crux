package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/cruxctl/crux/internal/mcp"
	"github.com/cruxctl/crux/internal/mcp/adapters"
)

func main() {
	storeRoot := os.Getenv("CRUX_MCP_STORE")
	if storeRoot == "" {
		home, _ := os.UserHomeDir()
		storeRoot = filepath.Join(home, ".crux-mcp", "store")
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	store := mcp.NewJsonStore(storeRoot)
	if err := store.Init(); err != nil {
		logger.Error("store init failed", "error", err)
		os.Exit(1)
	}

	registry := adapters.DefaultRegistry()
	watcher := mcp.NewWatcher(store, registry, logger)
	if err := watcher.Start(); err != nil {
		logger.Error("watcher start failed", "error", err)
		os.Exit(1)
	}

	server := mcp.NewServer(store, watcher, logger)
	if err := server.Run(); err != nil {
		logger.Error("server exited", "error", err)
		os.Exit(1)
	}
}
