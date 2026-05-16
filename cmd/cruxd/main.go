package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/cruxctl/crux/internal/config"
	"github.com/cruxctl/crux/internal/daemon"
)

func main() {
	var configPath, address, storePath, apiKey string
	var port int
	fs := flag.NewFlagSet("cruxd", flag.ExitOnError)
	fs.StringVar(&configPath, "config", "", "config YAML path")
	fs.StringVar(&address, "address", "", "listen address override")
	fs.IntVar(&port, "port", 0, "listen port override")
	fs.StringVar(&storePath, "store", "", "state store path override")
	fs.StringVar(&apiKey, "api-key", "", "API key override")
	fs.Parse(os.Args[1:])

	cfg, err := config.LoadDaemonConfig(configPath)
	if err != nil {
		fatal(err)
	}
	if address != "" {
		cfg.Server.Address = address
	}
	if port != 0 {
		cfg.Server.Port = port
	}
	if storePath != "" {
		cfg.Store.Path, err = config.ExpandPath(storePath)
		if err != nil {
			fatal(err)
		}
	}
	if apiKey != "" {
		cfg.Security.APIKey = apiKey
	}
	if err := cfg.Validate(); err != nil {
		fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
	if err := daemon.Run(context.Background(), cfg, logger); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
