package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cruxctl/crux/internal/client"
	"github.com/cruxctl/crux/internal/config"
	"github.com/cruxctl/cruxd/pkg/cruxapi"
	"gopkg.in/yaml.v3"
)

type CLI struct {
	out io.Writer
	err io.Writer
}

type rootOptions struct {
	configPath string
	context    string
	serverURL  string
	apiKey     string
	output     string
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
	return &CLI{out: out, err: err}
}

func (c *CLI) Run(ctx context.Context, args []string) int {
	opts, cmd, rest, err := parseRoot(args)
	if err != nil {
		fmt.Fprintln(c.err, err)
		c.usage()
		return 2
	}
	if cmd == "" || cmd == "help" || cmd == "--help" || cmd == "-h" {
		c.usage()
		return 0
	}

	var runErr error
	switch cmd {
	case "up":
		runErr = c.up(ctx, opts, rest)
	case "doctor":
		runErr = c.doctor(ctx, opts)
	case "version":
		runErr = c.version(ctx, opts)
	case "context":
		runErr = c.context(opts, rest)
	case "config":
		runErr = c.runtimeConfig(ctx, opts, rest)
	case "agents":
		runErr = c.agents(ctx, opts, rest)
	case "discover":
		runErr = c.discover(ctx, opts)
	case "run":
		runErr = c.runExecution(ctx, opts, rest)
	case "ps":
		runErr = c.ps(ctx, opts)
	case "trace":
		runErr = c.trace(ctx, opts, rest)
	case "events":
		runErr = c.events(ctx, opts)
	default:
		runErr = fmt.Errorf("unknown command %q", cmd)
	}
	if runErr != nil {
		fmt.Fprintln(c.err, runErr)
		return 1
	}
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

Commands:
  up                 Ensure cruxd is installed and running
  doctor             Check daemon health
  version            Print client and server version
  context            Manage CLI contexts
  config             Get or update runtime config
  discover           Discover managed CLI agents on PATH
  agents             Manage command-backed agents
  run                Run an agent
  ps                 List executions
  trace              Show events for an execution
  events             Show all daemon events`)
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
	if err := fs.Parse(args); err != nil {
		return opts, "", nil, err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return opts, "", nil, nil
	}
	return opts, remaining[0], remaining[1:], nil
}

func (c *CLI) up(ctx context.Context, opts rootOptions, args []string) error {
	var daemonConfigPath, address, storePath, apiKey, installScriptURL string
	var port int
	var yes, noStart bool
	fs := flag.NewFlagSet("up", flag.ContinueOnError)
	fs.SetOutput(c.err)
	fs.StringVar(&daemonConfigPath, "daemon-config", "", "daemon config YAML path")
	fs.StringVar(&address, "address", "", "listen address override")
	fs.IntVar(&port, "port", 0, "listen port override")
	fs.StringVar(&storePath, "store", "", "state store path override")
	fs.StringVar(&apiKey, "api-key", "", "API key override")
	fs.BoolVar(&yes, "yes", false, "download and install cruxd without prompting")
	fs.BoolVar(&yes, "y", false, "download and install cruxd without prompting")
	fs.BoolVar(&noStart, "no-start", false, "install cruxd but do not start it")
	fs.StringVar(&installScriptURL, "install-script-url", defaultInstallScriptURL, "cruxd install script URL")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cl, ctxCfg, err := c.client(opts)
	if err != nil {
		return err
	}
	healthCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	err = cl.Health(healthCtx)
	cancel()
	if err == nil {
		fmt.Fprintf(c.out, "cruxd: already running at %s\n", ctxCfg.ServerURL)
		return nil
	}

	daemonPath, found := findCruxd()
	if !found {
		if !yes {
			ok, err := c.confirm(fmt.Sprintf("cruxd is not installed or not on PATH. Download and run %s?", installScriptURL))
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("cruxd installation declined")
			}
		}
		if err := c.installCruxd(ctx, installScriptURL); err != nil {
			return err
		}
		if noStart {
			fmt.Fprintln(c.out, "cruxd installed")
			return nil
		}
		if err := waitForHealth(ctx, cl, postInstallHealthTimeout); err == nil {
			fmt.Fprintf(c.out, "cruxd: installed and running at %s\n", ctxCfg.ServerURL)
			return nil
		}
		daemonPath, found = findCruxd()
		if !found {
			return fmt.Errorf("cruxd install completed but no cruxd binary was found on PATH or ~/.local/bin")
		}
	}

	if noStart {
		return fmt.Errorf("cruxd is offline at %s", ctxCfg.ServerURL)
	}
	daemonArgs := buildCruxdArgs(daemonConfigPath, address, port, storePath, apiKey)
	return c.execCruxd(ctx, daemonPath, daemonArgs)
}

func buildCruxdArgs(configPath, address string, port int, storePath, apiKey string) []string {
	args := []string{}
	if configPath != "" {
		args = append(args, "--config", configPath)
	}
	if address != "" {
		args = append(args, "--address", address)
	}
	if port != 0 {
		args = append(args, "--port", fmt.Sprintf("%d", port))
	}
	if storePath != "" {
		args = append(args, "--store", storePath)
	}
	if apiKey != "" {
		args = append(args, "--api-key", apiKey)
	}
	return args
}

func (c *CLI) doctor(ctx context.Context, opts rootOptions) error {
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

func (c *CLI) version(ctx context.Context, opts rootOptions) error {
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
		for name, ctx := range cfg.Contexts {
			marker := " "
			if name == cfg.CurrentContext {
				marker = "*"
			}
			fmt.Fprintf(c.out, "%s %-16s %s\n", marker, name, ctx.ServerURL)
		}
	case "current":
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
	if cmdPath == "" {
		return fmt.Errorf("--cmd is required")
	}
	agent := cruxapi.Agent{
		Name:        args[0],
		Description: description,
		Command: cruxapi.CommandSpec{
			Path:           cmdPath,
			Args:           cmdArgs,
			Env:            parseEnv(envFlags),
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

func (c *CLI) discover(ctx context.Context, opts rootOptions) error {
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

func (c *CLI) ps(ctx context.Context, opts rootOptions) error {
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
	fmt.Fprintf(c.out, "%-29s %-18s %-10s %s\n", "ID", "AGENT", "STATUS", "QUEUED")
	for _, execution := range executions {
		fmt.Fprintf(c.out, "%-29s %-18s %-10s %s\n", execution.ID, execution.AgentName, execution.Status, execution.QueuedAt.Format("2006-01-02T15:04:05Z"))
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
	for _, event := range events {
		fmt.Fprintf(c.out, "%s %-20s %s\n", event.CreatedAt.Format("15:04:05"), event.Type, event.Message)
	}
	return nil
}

func (c *CLI) events(ctx context.Context, opts rootOptions) error {
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
	for _, event := range events {
		target := event.AgentName
		if event.ExecutionID != "" {
			target = event.ExecutionID
		}
		fmt.Fprintf(c.out, "%s %-29s %-22s %s\n", event.CreatedAt.Format("15:04:05"), target, event.Type, event.Message)
	}
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

func parseEnv(values []string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	env := map[string]string{}
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if !ok {
			continue
		}
		env[key] = val
	}
	return env
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
