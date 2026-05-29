package commands

import (
	"context"
	"fmt"
	"sort"

	"github.com/cruxctl/crux/internal/config"
)

func contextCmd() *Cmd {
	return &Cmd{
		Name:  "context",
		Short: "Manage CLI contexts",
		Subcommands: []*Cmd{
			{Name: "ls", Run: contextLs},
			{Name: "current", Run: contextCurrent},
			{Name: "use", Run: contextUse},
			{Name: "set", Run: contextSet},
		},
	}
}

func contextLs(ctx context.Context, args []string, opts Options) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: crux context ls")
	}
	cfg, _, err := config.LoadCLIConfig(opts.ConfigPath)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(cfg.Contexts))
	for name := range cfg.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	if opts.Format != "table" {
		rows := make([]struct {
			Name      string `json:"name" yaml:"name"`
			ServerURL string `json:"serverUrl" yaml:"serverUrl"`
			Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
			Current   bool   `json:"current" yaml:"current"`
		}, 0, len(names))
		for _, name := range names {
			ctx := cfg.Contexts[name]
			rows = append(rows, struct {
				Name      string `json:"name" yaml:"name"`
				ServerURL string `json:"serverUrl" yaml:"serverUrl"`
				Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
				Current   bool   `json:"current" yaml:"current"`
			}{Name: name, ServerURL: ctx.ServerURL, Namespace: ctx.Namespace, Current: name == cfg.CurrentContext})
		}
		return printOutput(opts, rows)
	}
	for _, name := range names {
		ctx := cfg.Contexts[name]
		marker := " "
		if name == cfg.CurrentContext {
			marker = "*"
		}
		fmt.Fprintf(opts.Out, "%s %-16s %s\n", marker, name, ctx.ServerURL)
	}
	return nil
}

func contextCurrent(ctx context.Context, args []string, opts Options) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: crux context current")
	}
	cfg, _, err := config.LoadCLIConfig(opts.ConfigPath)
	if err != nil {
		return err
	}
	if opts.Format != "table" {
		ctx, ok := cfg.Contexts[cfg.CurrentContext]
		if !ok {
			return fmt.Errorf("current context %q not found", cfg.CurrentContext)
		}
		return printOutput(opts, struct {
			Name      string `json:"name" yaml:"name"`
			ServerURL string `json:"serverUrl" yaml:"serverUrl"`
			Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
		}{Name: cfg.CurrentContext, ServerURL: ctx.ServerURL, Namespace: ctx.Namespace})
	}
	fmt.Fprintln(opts.Out, cfg.CurrentContext)
	return nil
}

func contextUse(ctx context.Context, args []string, opts Options) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: crux context use <name>")
	}
	cfg, path, err := config.LoadCLIConfig(opts.ConfigPath)
	if err != nil {
		return err
	}
	cctx, ok := cfg.Contexts[args[0]]
	if !ok {
		return fmt.Errorf("context %q not found", args[0])
	}
	cfg.CurrentContext = args[0]
	if err := config.SaveCLIConfig(path, cfg); err != nil {
		return err
	}
	if opts.Format != "table" {
		return printOutput(opts, struct {
			Name      string `json:"name" yaml:"name"`
			ServerURL string `json:"serverUrl" yaml:"serverUrl"`
			Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
			Current   bool   `json:"current" yaml:"current"`
		}{Name: args[0], ServerURL: cctx.ServerURL, Namespace: cctx.Namespace, Current: true})
	}
	fmt.Fprintf(opts.Out, "current context: %s\n", args[0])
	return nil
}

func contextSet(ctx context.Context, args []string, opts Options) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: crux context set <name> --server URL [--api-key KEY] [--namespace NS]")
	}
	var serverURL, apiKey, namespace string
	// Manual flag parsing since args are passed through
	rest := args[1:]
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--server":
			i++
			if i >= len(rest) {
				return fmt.Errorf("--server requires a value")
			}
			serverURL = rest[i]
		case "--api-key":
			i++
			if i >= len(rest) {
				return fmt.Errorf("--api-key requires a value")
			}
			apiKey = rest[i]
		case "--namespace":
			i++
			if i >= len(rest) {
				return fmt.Errorf("--namespace requires a value")
			}
			namespace = rest[i]
		default:
			return fmt.Errorf("unknown flag %q", rest[i])
		}
	}
	if serverURL == "" {
		return fmt.Errorf("--server is required")
	}
	cfg, path, err := config.LoadCLIConfig(opts.ConfigPath)
	if err != nil {
		return err
	}
	cfg.Contexts[args[0]] = config.CLIContext{ServerURL: serverURL, APIKey: apiKey, Namespace: namespace}
	if cfg.CurrentContext == "" {
		cfg.CurrentContext = args[0]
	}
	if err := config.SaveCLIConfig(path, cfg); err != nil {
		return err
	}
	if opts.Format != "table" {
		return printOutput(opts, struct {
			Name      string `json:"name" yaml:"name"`
			ServerURL string `json:"serverUrl" yaml:"serverUrl"`
			Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
			Current   bool   `json:"current" yaml:"current"`
		}{Name: args[0], ServerURL: serverURL, Namespace: namespace, Current: cfg.CurrentContext == args[0]})
	}
	fmt.Fprintf(opts.Out, "context %q set\n", args[0])
	return nil
}
