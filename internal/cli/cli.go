package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cruxctl/crux/internal/client"
	"github.com/cruxctl/crux/internal/config"
	"github.com/cruxctl/crux/internal/logging"
	"github.com/cruxctl/cruxd/pkg/cruxapi"
	"gopkg.in/yaml.v3"
)

type CLI struct {
	out    io.Writer
	err    io.Writer
	logger *slog.Logger
}

type rootOptions struct {
	configPath string
	context    string
	serverURL  string
	apiKey     string
	output     string
	logLevel   string
	logFile    string
	logFormat  string
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func New(out, err io.Writer) *CLI {
	return &CLI{out: out, err: err, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func (c *CLI) Run(ctx context.Context, args []string) int {
	opts, cmd, rest, err := parseRoot(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			c.usage()
			return 0
		}
		fmt.Fprintln(c.err, err)
		c.usage()
		return 2
	}
	if cmd == "" || cmd == "--help" || cmd == "-h" {
		c.usage()
		return 0
	}
	if cmd == "help" {
		if len(rest) == 0 {
			c.usage()
			return 0
		}
		if !c.commandUsage(rest[0], rest[1:]) {
			fmt.Fprintf(c.err, "unknown command %q\n", rest[0])
			return 1
		}
		return 0
	}
	if helpArg(rest) {
		if !c.commandUsage(cmd, rest[1:]) {
			fmt.Fprintf(c.err, "unknown command %q\n", cmd)
			return 1
		}
		return 0
	}
	if nestedHelpArg(cmd, rest) {
		if !c.commandUsage(cmd, []string{rest[0]}) {
			fmt.Fprintf(c.err, "unknown command %q\n", cmd)
			return 1
		}
		return 0
	}
	closeLogs, err := c.configureLogging(opts)
	if err != nil {
		fmt.Fprintln(c.err, err)
		return 1
	}
	defer closeLogs()
	c.logger.Info("command started", "command", cmd)

	var runErr error
	switch cmd {
	case "update":
		runErr = c.update(ctx, opts, rest)
	case "doctor":
		runErr = c.doctor(ctx, opts, rest)
	case "version":
		runErr = c.version(ctx, opts, rest)
	case "context":
		runErr = c.context(opts, rest)
	case "config":
		runErr = c.runtimeConfig(ctx, opts, rest)
	case "agents":
		runErr = c.agents(ctx, opts, rest)
	case "agent":
		runErr = c.agent(ctx, opts, rest)
	case "discover":
		runErr = c.discover(ctx, opts, rest)
	case "run":
		runErr = c.runExecution(ctx, opts, rest)
	case "ps":
		runErr = c.ps(ctx, opts, rest)
	case "trace":
		runErr = c.trace(ctx, opts, rest)
	case "events":
		runErr = c.events(ctx, opts, rest)
	default:
		runErr = fmt.Errorf("unknown command %q", cmd)
	}
	if runErr != nil {
		if errors.Is(runErr, flag.ErrHelp) {
			return 0
		}
		c.logger.Error("command failed", "command", cmd, "error", runErr)
		fmt.Fprintln(c.err, runErr)
		return 1
	}
	c.logger.Info("command finished", "command", cmd)
	return 0
}

func (c *CLI) usage() {
	fmt.Fprintln(c.out, `Crux Control MVP

Usage:
  crux [global flags] <command> [args]

Global flags:
  --config PATH      CLI config file (default ~/.config/crux/config.yaml)
  --context NAME     CLI context name
  --server URL       cruxd server URL override
  --api-key KEY      API key override
  -o, --output FMT   table, json, or yaml
  --log-level LEVEL  debug, info, warn, or error
  --log-file PATH    CLI log file override; "none" disables file logging

Commands:
  update             Install or update crux and cruxd
  doctor             Check daemon health
  version            Print client and server version
  context            Manage CLI contexts
  config             Get or update runtime config
  discover           Discover managed CLI agents on daemon host
  agents             Manage command-backed agents
  agent              Inspect one agent
  run                Run an agent
  ps                 List executions
  trace              Show events for an execution
  events             Show all daemon events`)
}

func (c *CLI) commandUsage(command string, args []string) bool {
	switch command {
	case "update":
		fmt.Fprintln(c.out, `Usage:
  crux update [--component all|crux|cruxd] [--version VERSION]
              [--crux-version VERSION] [--cruxd-version VERSION]
              [--force|-f] [--yes|-y] [--no-start]

Install or update crux and cruxd. --force purges and reinstalls the local cruxd binary and user service.`)
	case "doctor":
		fmt.Fprintln(c.out, `Usage:
  crux doctor

Check daemon health.`)
	case "version":
		fmt.Fprintln(c.out, `Usage:
  crux version

Print client and server version.`)
	case "context":
		if len(args) > 0 {
			return c.contextUsage(args[0])
		}
		fmt.Fprintln(c.out, `Usage:
  crux context ls
  crux context current
  crux context set <name> --server URL [--api-key KEY] [--namespace NS]
  crux context use <name>

Manage CLI contexts.`)
	case "config":
		if len(args) > 0 {
			return c.configUsage(args[0])
		}
		fmt.Fprintln(c.out, `Usage:
  crux config get
  crux config set [--concurrency N] [--job-timeout SECONDS]
                  [--max-output-bytes BYTES] [--discovery-timeout SECONDS]
                  [--trace-retention N] [--log-level LEVEL]
                  [--namespace NAME] [--allow-shell=true|false]

Get or update runtime config.`)
	case "discover":
		fmt.Fprintln(c.out, `Usage:
  crux discover

Discover managed CLI agents on the daemon host.`)
	case "agents":
		if len(args) > 0 {
			return c.agentsUsage(args[0])
		}
		fmt.Fprintln(c.out, `Usage:
  crux agents ls
  crux agents add <name> --cmd PATH [--arg ARG] [--env KEY=VALUE]
  crux agents rm <name>
  crux agents describe <name>

Manage command-backed agents.`)
	case "agent":
		if len(args) > 1 && args[1] == "usage" {
			fmt.Fprintln(c.out, `Usage:
  crux agent <name> usage

Show local execution usage metrics for one agent.`)
			return true
		}
		fmt.Fprintln(c.out, `Usage:
  crux agent <name> usage

Inspect one agent.`)
	case "run":
		fmt.Fprintln(c.out, `Usage:
  crux run <agent> <prompt> [--async]

Run an agent.`)
	case "ps":
		fmt.Fprintln(c.out, `Usage:
  crux ps

List executions.`)
	case "trace":
		fmt.Fprintln(c.out, `Usage:
  crux trace <execution-id|last>

Show events for an execution.`)
	case "events":
		fmt.Fprintln(c.out, `Usage:
  crux events [ls]

Show all daemon events.`)
	default:
		return false
	}
	return true
}

func (c *CLI) contextUsage(command string) bool {
	switch command {
	case "ls":
		fmt.Fprintln(c.out, "Usage:\n  crux context ls")
	case "current":
		fmt.Fprintln(c.out, "Usage:\n  crux context current")
	case "set":
		fmt.Fprintln(c.out, "Usage:\n  crux context set <name> --server URL [--api-key KEY] [--namespace NS]")
	case "use":
		fmt.Fprintln(c.out, "Usage:\n  crux context use <name>")
	default:
		return false
	}
	return true
}

func (c *CLI) configUsage(command string) bool {
	switch command {
	case "get":
		fmt.Fprintln(c.out, "Usage:\n  crux config get")
	case "set":
		fmt.Fprintln(c.out, `Usage:
  crux config set [--concurrency N] [--job-timeout SECONDS]
                  [--max-output-bytes BYTES] [--discovery-timeout SECONDS]
                  [--trace-retention N] [--log-level LEVEL]
                  [--namespace NAME] [--allow-shell=true|false]`)
	default:
		return false
	}
	return true
}

func (c *CLI) agentsUsage(command string) bool {
	switch command {
	case "ls":
		fmt.Fprintln(c.out, "Usage:\n  crux agents ls")
	case "add":
		fmt.Fprintln(c.out, "Usage:\n  crux agents add <name> --cmd PATH [--arg ARG] [--env KEY=VALUE] [--description TEXT] [--workdir DIR] [--timeout SECONDS]")
	case "rm":
		fmt.Fprintln(c.out, "Usage:\n  crux agents rm <name>")
	case "describe":
		fmt.Fprintln(c.out, "Usage:\n  crux agents describe <name>")
	default:
		return false
	}
	return true
}

func helpArg(args []string) bool {
	return len(args) > 0 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help")
}

func nestedHelpArg(command string, args []string) bool {
	if len(args) < 2 || (args[1] != "--help" && args[1] != "-h" && args[1] != "help") {
		return false
	}
	switch command {
	case "agent", "agents", "config", "context":
		return true
	default:
		return false
	}
}

func parseRoot(args []string) (rootOptions, string, []string, error) {
	var opts rootOptions
	fs := flag.NewFlagSet("crux", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.configPath, "config", "", "CLI config path")
	fs.StringVar(&opts.context, "context", "", "context name")
	fs.StringVar(&opts.serverURL, "server", "", "server URL")
	fs.StringVar(&opts.apiKey, "api-key", "", "API key")
	fs.StringVar(&opts.output, "output", "table", "output format")
	fs.StringVar(&opts.output, "o", "table", "output format")
	fs.StringVar(&opts.logLevel, "log-level", "", "log level")
	fs.StringVar(&opts.logFile, "log-file", "", "log file")
	fs.StringVar(&opts.logFormat, "log-format", "", "log format")
	if err := fs.Parse(args); err != nil {
		return opts, "", nil, err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return opts, "", nil, nil
	}
	return opts, remaining[0], remaining[1:], nil
}

func (c *CLI) update(ctx context.Context, opts rootOptions, args []string) error {
	var component, version, cruxVersion, cruxdVersion, cruxScriptURL, cruxPowerShellURL, cruxdScriptURL, cruxdPowerShellURL string
	var yes, force, noStart bool
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(c.err)
	fs.StringVar(&component, "component", "all", "component to update: all, crux, or cruxd")
	fs.StringVar(&version, "version", "latest", "version for both components")
	fs.StringVar(&cruxVersion, "crux-version", "", "crux CLI version override")
	fs.StringVar(&cruxdVersion, "cruxd-version", "", "cruxd version override")
	fs.BoolVar(&yes, "yes", false, "run without prompting")
	fs.BoolVar(&yes, "y", false, "run without prompting")
	fs.BoolVar(&force, "force", false, "purge and reinstall cruxd binary and service")
	fs.BoolVar(&force, "f", false, "purge and reinstall cruxd binary and service")
	fs.BoolVar(&noStart, "no-start", false, "install cruxd but do not start it")
	fs.StringVar(&cruxScriptURL, "crux-script-url", defaultCruxInstallScriptURL, "crux shell install script URL")
	fs.StringVar(&cruxPowerShellURL, "crux-powershell-url", defaultCruxInstallPowerShellURL, "crux PowerShell install script URL")
	fs.StringVar(&cruxdScriptURL, "cruxd-script-url", defaultCruxdInstallScriptURL, "cruxd shell install script URL")
	fs.StringVar(&cruxdPowerShellURL, "cruxd-powershell-url", defaultCruxdInstallPowerShellURL, "cruxd PowerShell install script URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: crux update [flags]")
	}
	component = strings.ToLower(strings.TrimSpace(component))
	if component != "all" && component != "crux" && component != "cruxd" {
		return fmt.Errorf("--component must be all, crux, or cruxd")
	}
	if cruxVersion == "" {
		cruxVersion = version
	}
	if cruxdVersion == "" {
		cruxdVersion = version
	}
	if force && !yes {
		ok, err := c.confirm("Force reinstall will remove the local cruxd binary and user service before reinstalling. Continue?")
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("update declined")
		}
	}
	env := map[string]string{
		"CRUX_VERSION":  cruxVersion,
		"CRUXD_VERSION": cruxdVersion,
	}
	installerArgs := []string{"--version", cruxVersion}
	if noStart {
		installerArgs = append(installerArgs, "--no-start")
	}
	if force {
		installerArgs = append(installerArgs, "--force")
	}
	switch component {
	case "all":
		fmt.Fprintln(c.out, "updating crux and cruxd")
		if err := c.runInstaller(ctx, cruxScriptURL, cruxPowerShellURL, installerArgs, env); err != nil {
			return err
		}
	case "crux":
		fmt.Fprintln(c.out, "updating crux")
		args := append([]string{}, installerArgs...)
		args = append(args, "--skip-cruxd")
		if err := c.runInstaller(ctx, cruxScriptURL, cruxPowerShellURL, args, env); err != nil {
			return err
		}
	case "cruxd":
		fmt.Fprintln(c.out, "updating cruxd")
		args := []string{"--version", cruxdVersion}
		if noStart {
			args = append(args, "--no-start")
		}
		if force {
			args = append(args, "--force")
		}
		if err := c.runInstaller(ctx, cruxdScriptURL, cruxdPowerShellURL, args, env); err != nil {
			return err
		}
	}
	if noStart || component == "crux" {
		return nil
	}
	cl, ctxCfg, err := c.client(opts)
	if err != nil {
		return err
	}
	if err := waitForHealth(ctx, cl, postInstallHealthTimeout); err != nil {
		return fmt.Errorf("cruxd was updated but did not become healthy at %s: %w", ctxCfg.ServerURL, err)
	}
	fmt.Fprintf(c.out, "cruxd: running at %s\n", ctxCfg.ServerURL)
	return nil
}

func (c *CLI) doctor(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: crux doctor")
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	if err := cl.Health(ctx); err != nil {
		return fmt.Errorf("cruxd health: %w", err)
	}
	version, err := cl.Version(ctx)
	if err != nil {
		return fmt.Errorf("cruxd version: %w", err)
	}
	fmt.Fprintf(c.out, "cruxd: ok (%s)\n", version)
	return nil
}

func (c *CLI) version(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: crux version")
	}
	fmt.Fprintf(c.out, "crux client: %s\n", cruxapi.Version)
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	version, err := cl.Version(ctx)
	if err != nil {
		fmt.Fprintf(c.out, "crux server: unavailable (%v)\n", err)
		return nil
	}
	fmt.Fprintf(c.out, "crux server: %s\n", version)
	return nil
}

func (c *CLI) context(opts rootOptions, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: crux context <ls|current|set|use>")
	}
	cfg, path, err := config.LoadCLIConfig(opts.configPath)
	if err != nil {
		return err
	}
	switch args[0] {
	case "ls":
		if len(args) != 1 {
			return fmt.Errorf("usage: crux context ls")
		}
		names := make([]string, 0, len(cfg.Contexts))
		for name := range cfg.Contexts {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			ctx := cfg.Contexts[name]
			marker := " "
			if name == cfg.CurrentContext {
				marker = "*"
			}
			fmt.Fprintf(c.out, "%s %-16s %s\n", marker, name, ctx.ServerURL)
		}
	case "current":
		if len(args) != 1 {
			return fmt.Errorf("usage: crux context current")
		}
		fmt.Fprintln(c.out, cfg.CurrentContext)
	case "use":
		if len(args) != 2 {
			return fmt.Errorf("usage: crux context use <name>")
		}
		if _, ok := cfg.Contexts[args[1]]; !ok {
			return fmt.Errorf("context %q not found", args[1])
		}
		cfg.CurrentContext = args[1]
		return config.SaveCLIConfig(path, cfg)
	case "set":
		if len(args) < 2 {
			return fmt.Errorf("usage: crux context set <name> --server URL [--api-key KEY] [--namespace NS]")
		}
		var serverURL, apiKey, namespace string
		fs := flag.NewFlagSet("context set", flag.ContinueOnError)
		fs.SetOutput(c.err)
		fs.StringVar(&serverURL, "server", "", "server URL")
		fs.StringVar(&apiKey, "api-key", "", "API key")
		fs.StringVar(&namespace, "namespace", "default", "namespace")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if fs.NArg() != 0 {
			return fmt.Errorf("usage: crux context set <name> --server URL [--api-key KEY] [--namespace NS]")
		}
		if serverURL == "" {
			return fmt.Errorf("--server is required")
		}
		cfg.Contexts[args[1]] = config.CLIContext{ServerURL: serverURL, APIKey: apiKey, Namespace: namespace}
		if cfg.CurrentContext == "" {
			cfg.CurrentContext = args[1]
		}
		return config.SaveCLIConfig(path, cfg)
	default:
		return fmt.Errorf("unknown context command %q", args[0])
	}
	return nil
}

func (c *CLI) runtimeConfig(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: crux config <get|set>")
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	switch args[0] {
	case "get":
		if len(args) != 1 {
			return fmt.Errorf("usage: crux config get")
		}
		runtime, err := cl.RuntimeConfig(ctx)
		if err != nil {
			return err
		}
		return c.print(opts.output, runtime)
	case "set":
		patch, err := parseRuntimePatch(args[1:], c.err)
		if err != nil {
			return err
		}
		runtime, err := cl.UpdateRuntimeConfig(ctx, patch)
		if err != nil {
			return err
		}
		return c.print(opts.output, runtime)
	default:
		return fmt.Errorf("unknown config command %q", args[0])
	}
}

func parseRuntimePatch(args []string, errOut io.Writer) (cruxapi.RuntimeConfigPatch, error) {
	var concurrency, timeout, maxOutput, discoveryTimeout, retention int
	var logLevel, namespace string
	var allowShell bool
	fs := flag.NewFlagSet("config set", flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.IntVar(&concurrency, "concurrency", 0, "worker concurrency")
	fs.IntVar(&timeout, "job-timeout", 0, "job timeout seconds")
	fs.IntVar(&maxOutput, "max-output-bytes", 0, "max output bytes")
	fs.IntVar(&discoveryTimeout, "discovery-timeout", 0, "discovery timeout seconds")
	fs.IntVar(&retention, "trace-retention", 0, "trace retention entries")
	fs.StringVar(&logLevel, "log-level", "", "log level")
	fs.StringVar(&namespace, "namespace", "", "default namespace")
	fs.BoolVar(&allowShell, "allow-shell", false, "allow shell-backed agents")
	if err := fs.Parse(args); err != nil {
		return cruxapi.RuntimeConfigPatch{}, err
	}
	if fs.NArg() != 0 {
		return cruxapi.RuntimeConfigPatch{}, fmt.Errorf("usage: crux config set [flags]")
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
	if flagWasSet(fs, "allow-shell") {
		patch.AllowShellCommands = &allowShell
	}
	return patch, nil
}

func (c *CLI) agents(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: crux agents <ls|add|rm|describe>")
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	switch args[0] {
	case "ls":
		if len(args) != 1 {
			return fmt.Errorf("usage: crux agents ls")
		}
		agents, err := cl.ListAgents(ctx)
		if err != nil {
			return err
		}
		if opts.output != "table" {
			return c.print(opts.output, agents)
		}
		fmt.Fprintf(c.out, "%-18s %-10s %s\n", "NAME", "STATUS", "COMMAND")
		for _, agent := range agents {
			fmt.Fprintf(c.out, "%-18s %-10s %s %s\n", agent.Name, agent.Status, agent.Command.Path, strings.Join(agent.Command.Args, " "))
		}
	case "describe":
		if len(args) != 2 {
			return fmt.Errorf("usage: crux agents describe <name>")
		}
		agents, err := cl.ListAgents(ctx)
		if err != nil {
			return err
		}
		for _, agent := range agents {
			if agent.Name == cruxapi.CleanAgentName(args[1]) {
				return c.print(firstNonEmpty(opts.output, "yaml"), agent)
			}
		}
		return fmt.Errorf("agent %q not found", args[1])
	case "add":
		return c.addAgent(ctx, opts, cl, args[1:])
	case "rm":
		if len(args) != 2 {
			return fmt.Errorf("usage: crux agents rm <name>")
		}
		return cl.DeleteAgent(ctx, args[1])
	default:
		return fmt.Errorf("unknown agents command %q", args[0])
	}
	return nil
}

func (c *CLI) agent(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) == 0 || helpArg(args) {
		return fmt.Errorf("usage: crux agent <name> usage")
	}
	if len(args) == 3 && (args[2] == "--help" || args[2] == "-h" || args[2] == "help") && args[1] == "usage" {
		c.commandUsage("agent", []string{args[0], "usage"})
		return nil
	}
	if len(args) != 2 || args[1] != "usage" {
		return fmt.Errorf("usage: crux agent <name> usage")
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	usage, err := cl.AgentUsage(ctx, args[0])
	if err != nil {
		return err
	}
	if opts.output != "table" {
		return c.print(opts.output, usage)
	}
	return c.printAgentUsage(usage)
}

func (c *CLI) addAgent(ctx context.Context, opts rootOptions, cl *client.Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: crux agents add <name> --cmd PATH [--arg ARG] [--env KEY=VALUE]")
	}
	var cmdPath, description, workdir string
	var cmdArgs, envFlags multiFlag
	var timeout int
	fs := flag.NewFlagSet("agents add", flag.ContinueOnError)
	fs.SetOutput(c.err)
	fs.StringVar(&cmdPath, "cmd", "", "command path")
	fs.Var(&cmdArgs, "arg", "command argument; repeatable")
	fs.Var(&envFlags, "env", "environment KEY=VALUE; repeatable")
	fs.StringVar(&description, "description", "", "description")
	fs.StringVar(&workdir, "workdir", "", "working directory")
	fs.IntVar(&timeout, "timeout", 0, "agent timeout seconds")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: crux agents add <name> --cmd PATH [--arg ARG] [--env KEY=VALUE]")
	}
	if cmdPath == "" {
		return fmt.Errorf("--cmd is required")
	}
	env, err := parseEnv(envFlags)
	if err != nil {
		return err
	}
	agent := cruxapi.Agent{
		Name:        args[0],
		Description: description,
		Command: cruxapi.CommandSpec{
			Path:           cmdPath,
			Args:           cmdArgs,
			Env:            env,
			WorkingDir:     workdir,
			TimeoutSeconds: timeout,
		},
		Status: cruxapi.AgentReady,
	}
	saved, err := cl.UpsertAgent(ctx, agent)
	if err != nil {
		return err
	}
	return c.print(opts.output, saved)
}

func (c *CLI) discover(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: crux discover")
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	results, err := cl.Discover(ctx)
	if err != nil {
		return err
	}
	if opts.output != "table" {
		return c.print(opts.output, results)
	}
	if len(results) == 0 {
		fmt.Fprintln(c.out, "no managed CLI agents found")
		return nil
	}
	fmt.Fprintf(c.out, "%-18s %-50s %s\n", "NAME", "COMMAND", "VERSION")
	for _, result := range results {
		fmt.Fprintf(c.out, "%-18s %-50s %s\n", result.Agent.Name, result.Agent.Command.Path, result.Version)
	}
	return nil
}

func (c *CLI) runExecution(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: crux run <agent> <prompt> [--async]")
	}
	async := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--async" {
			async = true
			continue
		}
		filtered = append(filtered, arg)
	}
	if len(filtered) < 2 {
		return fmt.Errorf("usage: crux run <agent> <prompt> [--async]")
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	execution, err := cl.Run(ctx, cruxapi.SubmitExecutionRequest{
		AgentName: filtered[0],
		Prompt:    strings.Join(filtered[1:], " "),
		Wait:      !async,
	})
	if err != nil {
		return err
	}
	if opts.output != "table" || async {
		return c.print(opts.output, execution)
	}
	fmt.Fprint(c.out, execution.Stdout)
	if execution.Stderr != "" {
		fmt.Fprint(c.err, execution.Stderr)
	}
	if execution.Status != cruxapi.ExecutionSucceeded {
		return fmt.Errorf("execution %s failed: %s", execution.ID, execution.Error)
	}
	return nil
}

func (c *CLI) ps(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: crux ps")
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	executions, err := cl.ListExecutions(ctx)
	if err != nil {
		return err
	}
	if opts.output != "table" {
		return c.print(opts.output, executions)
	}
	fmt.Fprintf(c.out, "%-29s %-18s %-10s %-10s %-6s %s\n", "ID", "AGENT", "STATUS", "DURATION", "EXIT", "QUEUED")
	for _, execution := range executions {
		fmt.Fprintf(c.out, "%-29s %-18s %-10s %-10s %-6d %s\n", execution.ID, execution.AgentName, execution.Status, executionDuration(execution), execution.ExitCode, execution.QueuedAt.Format(time.RFC3339))
	}
	return nil
}

func (c *CLI) trace(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: crux trace <execution-id|last>")
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	id := args[0]
	if id == "last" {
		executions, err := cl.ListExecutions(ctx)
		if err != nil {
			return err
		}
		if len(executions) == 0 {
			return fmt.Errorf("no executions found")
		}
		id = executions[0].ID
	}
	events, err := cl.Events(ctx, id)
	if err != nil {
		return err
	}
	if opts.output != "table" {
		return c.print(opts.output, events)
	}
	printEventTable(c.out, events, false)
	return nil
}

func (c *CLI) events(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) > 1 || (len(args) == 1 && args[0] != "ls") {
		return fmt.Errorf("usage: crux events [ls]")
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	events, err := cl.Events(ctx, "")
	if err != nil {
		return err
	}
	if opts.output != "table" {
		return c.print(opts.output, events)
	}
	printEventTable(c.out, events, true)
	return nil
}

func (c *CLI) client(opts rootOptions) (*client.Client, config.CLIContext, error) {
	cfg, _, err := config.LoadCLIConfig(opts.configPath)
	if err != nil {
		return nil, config.CLIContext{}, err
	}
	ctxCfg, _, err := cfg.ActiveContext(opts.context)
	if err != nil {
		return nil, config.CLIContext{}, err
	}
	if env := strings.TrimSpace(os.Getenv("CRUX_SERVER_URL")); env != "" {
		ctxCfg.ServerURL = env
	}
	if env := strings.TrimSpace(os.Getenv("CRUX_API_KEY")); env != "" {
		ctxCfg.APIKey = env
	}
	if opts.serverURL != "" {
		ctxCfg.ServerURL = opts.serverURL
	}
	if opts.apiKey != "" {
		ctxCfg.APIKey = opts.apiKey
	}
	return client.New(ctxCfg.ServerURL, ctxCfg.APIKey), ctxCfg, nil
}

func (c *CLI) configureLogging(opts rootOptions) (func() error, error) {
	logger, closeFn, err := logging.New(logging.Options{
		Level:      opts.logLevel,
		File:       opts.logFile,
		Format:     opts.logFormat,
		MaxSizeMB:  10,
		MaxBackups: 5,
	})
	if err != nil {
		return nil, err
	}
	c.logger = logger
	return closeFn, nil
}

func (c *CLI) print(format string, value any) error {
	switch format {
	case "", "table", "json":
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(c.out, string(data))
	case "yaml":
		data, err := yaml.Marshal(value)
		if err != nil {
			return err
		}
		fmt.Fprint(c.out, string(data))
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
	return nil
}

func (c *CLI) printAgentUsage(usage cruxapi.AgentUsage) error {
	fmt.Fprintf(c.out, "Agent: %s\n", usage.AgentName)
	fmt.Fprintf(c.out, "Executions: total=%d succeeded=%d failed=%d running=%d queued=%d canceled=%d successRate=%.1f%%\n",
		usage.ExecutionsTotal, usage.Succeeded, usage.Failed, usage.Running, usage.Queued, usage.Canceled, usage.SuccessRate*100)
	fmt.Fprintf(c.out, "Output: stdoutBytes=%d stderrBytes=%d errors=%d\n", usage.StdoutBytes, usage.StderrBytes, usage.ErrorCount)
	fmt.Fprintf(c.out, "Duration: total=%s average=%s max=%s\n",
		formatSeconds(usage.TotalDurationSeconds), formatSeconds(usage.AverageDurationSeconds), formatSeconds(usage.MaxDurationSeconds))
	if usage.LastExecutionID != "" {
		queued := "-"
		if usage.LastQueuedAt != nil {
			queued = usage.LastQueuedAt.Format(time.RFC3339)
		}
		fmt.Fprintf(c.out, "Last: id=%s status=%s exit=%d queued=%s\n",
			usage.LastExecutionID, usage.LastStatus, usage.LastExitCode, queued)
		if usage.LastError != "" {
			fmt.Fprintf(c.out, "LastError: %s\n", usage.LastError)
		}
	}
	if len(usage.ExitCodes) > 0 {
		fmt.Fprintln(c.out, "Exit codes:")
		codes := make([]string, 0, len(usage.ExitCodes))
		for code := range usage.ExitCodes {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		for _, code := range codes {
			count := usage.ExitCodes[code]
			fmt.Fprintf(c.out, "  %s: %d\n", code, count)
		}
	}
	if len(usage.EventCounts) > 0 {
		fmt.Fprintln(c.out, "Events:")
		eventTypes := make([]string, 0, len(usage.EventCounts))
		for eventType := range usage.EventCounts {
			eventTypes = append(eventTypes, string(eventType))
		}
		sort.Strings(eventTypes)
		for _, eventType := range eventTypes {
			count := usage.EventCounts[cruxapi.EventType(eventType)]
			fmt.Fprintf(c.out, "  %s: %d\n", eventType, count)
		}
	}
	if len(usage.ExternalMetrics) > 0 {
		fmt.Fprintln(c.out, "External metrics:")
		for _, metric := range usage.ExternalMetrics {
			status := "not available"
			if metric.Available {
				status = firstNonEmpty(metric.Value, "available")
			}
			fmt.Fprintf(c.out, "  %s: %s", metric.Name, status)
			if metric.Description != "" {
				fmt.Fprintf(c.out, " - %s", metric.Description)
			}
			fmt.Fprintln(c.out)
		}
	}
	for _, note := range usage.Notes {
		fmt.Fprintf(c.out, "Note: %s\n", note)
	}
	return nil
}

func parseEnv(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	env := map[string]string{}
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --env value %q; expected KEY=VALUE", value)
		}
		env[key] = val
	}
	return env, nil
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	wasSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			wasSet = true
		}
	})
	return wasSet
}

func firstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func printEventTable(out io.Writer, events []cruxapi.Event, includeTarget bool) {
	if includeTarget {
		fmt.Fprintf(out, "%-20s %-29s %-18s %-24s %-8s %s\n", "TIME", "EXECUTION", "AGENT", "TYPE", "EXIT", "MESSAGE")
	} else {
		fmt.Fprintf(out, "%-20s %-18s %-24s %-8s %s\n", "TIME", "AGENT", "TYPE", "EXIT", "MESSAGE")
	}
	for _, event := range events {
		exitCode := eventDataString(event, "exitCode")
		if includeTarget {
			fmt.Fprintf(out, "%-20s %-29s %-18s %-24s %-8s %s\n",
				event.CreatedAt.Format(time.RFC3339), firstNonEmpty(event.ExecutionID, "-"), firstNonEmpty(event.AgentName, "-"), event.Type, firstNonEmpty(exitCode, "-"), event.Message)
			continue
		}
		fmt.Fprintf(out, "%-20s %-18s %-24s %-8s %s\n",
			event.CreatedAt.Format(time.RFC3339), firstNonEmpty(event.AgentName, "-"), event.Type, firstNonEmpty(exitCode, "-"), event.Message)
	}
}

func eventDataString(event cruxapi.Event, key string) string {
	if event.Data == nil {
		return ""
	}
	value, ok := event.Data[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case int:
		return strconv.Itoa(typed)
	case float64:
		return strconv.Itoa(int(typed))
	default:
		return fmt.Sprint(typed)
	}
}

func executionDuration(execution cruxapi.Execution) string {
	if execution.StartedAt == nil {
		return "-"
	}
	end := execution.UpdatedAt
	if execution.CompletedAt != nil {
		end = *execution.CompletedAt
	}
	if end.Before(*execution.StartedAt) {
		return "-"
	}
	return end.Sub(*execution.StartedAt).Round(time.Millisecond).String()
}

func formatSeconds(seconds float64) string {
	if seconds <= 0 {
		return "0s"
	}
	return time.Duration(seconds * float64(time.Second)).Round(time.Millisecond).String()
}
