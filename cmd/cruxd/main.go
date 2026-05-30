package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/cruxctl/crux/internal/config"
	"github.com/cruxctl/crux/internal/daemon"
	"github.com/cruxctl/crux/internal/logging"
)

func main() {
	var configPath, address, storePath, apiKey, logLevel, logFile, logFormat string
	var port int
	fs := flag.NewFlagSet("cruxd", flag.ExitOnError)
	fs.StringVar(&configPath, "config", "", "config YAML path")
	fs.StringVar(&address, "address", "", "listen address override")
	fs.IntVar(&port, "port", 0, "listen port override")
	fs.StringVar(&storePath, "store", "", "state store root override")
	fs.StringVar(&apiKey, "api-key", "", "API key override")
	fs.StringVar(&logLevel, "log-level", "", "log level: debug, info, warn, or error")
	fs.StringVar(&logFile, "log-file", "", "log file path")
	fs.StringVar(&logFormat, "log-format", "", "log format: text or json")
	fs.Parse(os.Args[1:])

	cfg, err := config.Load(configPath)
	if err != nil {
		fatal(err)
	}
	if address != "" {
		cfg.Daemon.Host = address
	}
	if port != 0 {
		cfg.Daemon.Port = port
	}
	if storePath != "" {
		root, err := config.ExpandPath(storePath)
		if err != nil {
			fatal(err)
		}
		cfg.State.Root = root
	}
	if apiKey != "" {
		cfg.API.APIKeys = append(cfg.API.APIKeys, apiKey)
		if cfg.API.AuthMode == "none" {
			cfg.API.AuthMode = "api_key"
		}
	}
	if logLevel != "" {
		cfg.Daemon.LogLevel = logLevel
	}
	if logFile != "" {
		f, err := config.ExpandPath(logFile)
		if err != nil {
			fatal(err)
		}
		cfg.Daemon.LogFile = f
	}
	if logFormat != "" {
		cfg.Daemon.LogFormat = logFormat
	}
	if err := cfg.Validate(); err != nil {
		fatal(err)
	}

	logger, closeLogs, err := logging.New(logging.Options{
		Level:      cfg.Daemon.LogLevel,
		File:       cfg.Daemon.LogFile,
		Format:     cfg.Daemon.LogFormat,
		MaxSizeMB:  10,
		MaxBackups: 5,
	}, os.Stderr)
	if err != nil {
		fatal(err)
	}
	defer closeLogs()
	if err := daemon.Run(context.Background(), cfg, logger); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
