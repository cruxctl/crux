package commands

import (
	"context"
	"fmt"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

func configCmd() *Cmd {
	return &Cmd{
		Name:  "config",
		Short: "Get or update runtime config",
		Subcommands: []*Cmd{
			{Name: "get", Run: configGet},
			{Name: "set", Run: configSet},
		},
	}
}

func configGet(ctx context.Context, args []string, opts Options) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: crux config get")
	}
	cl, _, err := buildClient(opts)
	if err != nil {
		return err
	}
	runtime, err := cl.RuntimeConfig(ctx)
	if err != nil {
		return err
	}
	if opts.Format != "table" {
		return printOutput(opts, runtime)
	}
	printRuntimeConfigTable(opts.Out, runtime)
	return nil
}

func configSet(ctx context.Context, args []string, opts Options) error {
	patch, err := parseRuntimePatch(args)
	if err != nil {
		return err
	}
	cl, _, err := buildClient(opts)
	if err != nil {
		return err
	}
	runtime, err := cl.UpdateRuntimeConfig(ctx, patch)
	if err != nil {
		return err
	}
	if opts.Format != "table" {
		return printOutput(opts, runtime)
	}
	printRuntimeConfigTable(opts.Out, runtime)
	return nil
}

func parseRuntimePatch(args []string) (cruxapi.RuntimeConfigPatch, error) {
	var concurrency, timeout, maxOutput, discoveryTimeout, retention int
	var logLevel, namespace string
	var allowShell bool
	var hasAllowShell bool

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--concurrency":
			i++
			if i >= len(args) {
				return cruxapi.RuntimeConfigPatch{}, fmt.Errorf("--concurrency requires a value")
			}
			fmt.Sscanf(args[i], "%d", &concurrency)
		case "--job-timeout":
			i++
			if i >= len(args) {
				return cruxapi.RuntimeConfigPatch{}, fmt.Errorf("--job-timeout requires a value")
			}
			fmt.Sscanf(args[i], "%d", &timeout)
		case "--max-output-bytes":
			i++
			if i >= len(args) {
				return cruxapi.RuntimeConfigPatch{}, fmt.Errorf("--max-output-bytes requires a value")
			}
			fmt.Sscanf(args[i], "%d", &maxOutput)
		case "--discovery-timeout":
			i++
			if i >= len(args) {
				return cruxapi.RuntimeConfigPatch{}, fmt.Errorf("--discovery-timeout requires a value")
			}
			fmt.Sscanf(args[i], "%d", &discoveryTimeout)
		case "--trace-retention":
			i++
			if i >= len(args) {
				return cruxapi.RuntimeConfigPatch{}, fmt.Errorf("--trace-retention requires a value")
			}
			fmt.Sscanf(args[i], "%d", &retention)
		case "--log-level":
			i++
			if i >= len(args) {
				return cruxapi.RuntimeConfigPatch{}, fmt.Errorf("--log-level requires a value")
			}
			logLevel = args[i]
		case "--namespace":
			i++
			if i >= len(args) {
				return cruxapi.RuntimeConfigPatch{}, fmt.Errorf("--namespace requires a value")
			}
			namespace = args[i]
		case "--allow-shell":
			i++
			if i >= len(args) {
				return cruxapi.RuntimeConfigPatch{}, fmt.Errorf("--allow-shell requires a value")
			}
			allowShell = args[i] == "true"
			hasAllowShell = true
		default:
			return cruxapi.RuntimeConfigPatch{}, fmt.Errorf("unknown flag %q", args[i])
		}
	}

	patch := cruxapi.RuntimeConfigPatch{}
	if concurrency != 0 {
		patch.WorkerConcurrency = &concurrency
	}
	if timeout != 0 {
		patch.JobTimeoutSeconds = &timeout
	}
	if maxOutput != 0 {
		patch.MaxOutputBytes = &maxOutput
	}
	if discoveryTimeout != 0 {
		patch.DiscoveryTimeoutSecs = &discoveryTimeout
	}
	if retention != 0 {
		patch.TraceRetentionEntries = &retention
	}
	if logLevel != "" {
		patch.LogLevel = &logLevel
	}
	if namespace != "" {
		patch.DefaultNamespace = &namespace
	}
	if hasAllowShell {
		patch.AllowShellCommands = &allowShell
	}
	return patch, nil
}
