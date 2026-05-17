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
	"os/exec"
	"path/filepath"
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
	rest, err = applyOutputFlag(opts.output, rest, &opts.output)
	if err != nil {
		fmt.Fprintln(c.err, err)
		return 2
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
	if cmd == "agent" && len(rest) == 0 {
		c.commandUsage("agent", nil)
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
  crux <command> [args] [global flags]

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
  agents             Manage and monitor command-backed agents
  agent              Inspect and operate one agent
  run                Run an agent with resume, history, and fallback options
  ps                 List executions with filters
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
  crux agents usage
  crux agents cost
  crux agents sessions
  crux agent <name> describe
  crux agent <name> usage
  crux agent <name> cost
  crux agent <name> sessions
  crux agent <name> exec [--resume SESSION] [--send TEXT] [--expect TEXT]
  crux agent <name> history
  crux agent <name> rm

Manage and monitor command-backed agents.`)
	case "agent":
		if len(args) > 1 && args[1] == "describe" {
			fmt.Fprintln(c.out, `Usage:
  crux agent <name> describe

Describe one agent.`)
			return true
		}
		if len(args) > 1 && args[1] == "usage" {
			fmt.Fprintln(c.out, `Usage:
  crux agent <name> usage

Show local execution usage metrics for one agent.`)
			return true
		}
		if len(args) > 1 && args[1] == "cost" {
			fmt.Fprintln(c.out, `Usage:
  crux agent <name> cost

Show realtime local cost signals and provider usage evidence for one agent.`)
			return true
		}
		if len(args) > 1 && args[1] == "sessions" {
			fmt.Fprintln(c.out, `Usage:
  crux agent <name> sessions

List provider sessions when exposed and Crux-owned execution history sessions.`)
			return true
		}
		if len(args) > 1 && args[1] == "history" {
			fmt.Fprintln(c.out, `Usage:
  crux agent <name> history [ls]
  crux agent <name> history show <execution-id>
  crux agent <name> history share <execution-id> <target-agent> [--prompt TEXT]

View immutable execution history and replay/share edited context into another agent.`)
			return true
		}
		if len(args) > 1 && args[1] == "exec" {
			fmt.Fprintln(c.out, `Usage:
  crux agent <name> exec [--resume SESSION] [--workdir DIR]
                         [--send TEXT] [--expect TEXT]
                         [--driver auto|script|expect|direct]
                         [--timeout SECONDS] [--transcript PATH]
                         [--dry-run] [--no-record] [-- PROVIDER_ARGS...]

Open the provider TUI in a PTY, optionally drive input programmatically, capture a transcript, and record the result in Crux history.`)
			return true
		}
		if len(args) > 1 && args[1] == "resume" {
			fmt.Fprintln(c.out, `Usage:
  crux agent <name> resume <session-id|last> <prompt> [--async] [--fallback AGENT[,AGENT]]

Resume a provider session through the agent's managed CLI adapter.`)
			return true
		}
		if len(args) > 1 && args[1] == "fallback" {
			fmt.Fprintln(c.out, `Usage:
  crux agent <name> fallback
  crux agent <name> fallback set <agent>[,<agent>...]
  crux agent <name> fallback clear

View or set fallback agents used by `+"`crux run`"+` when the primary agent fails.`)
			return true
		}
		if len(args) > 1 && args[1] == "rm" {
			fmt.Fprintln(c.out, `Usage:
  crux agent <name> rm

Remove one agent.`)
			return true
		}
		fmt.Fprintln(c.out, `Usage:
  crux agent <name>
  crux agent <name> describe
  crux agent <name> usage
  crux agent <name> cost
  crux agent <name> sessions
  crux agent <name> exec
  crux agent <name> history [ls|show|share]
  crux agent <name> resume <session-id|last> <prompt>
  crux agent <name> fallback [set|clear]
  crux agent <name> rm

Inspect and operate one agent.`)
	case "run":
		fmt.Fprintln(c.out, `Usage:
  crux run <agent> <prompt> [--async] [--resume SESSION] [--fallback AGENT[,AGENT]]
  crux run <agent> --from EXECUTION_ID [--prompt TEXT] [--fallback AGENT[,AGENT]]

Run an agent. --from replays Crux history into a new run without mutating history.`)
	case "ps":
		fmt.Fprintln(c.out, `Usage:
  crux ps [--agent NAME] [--status STATUS] [--last N]

List executions with optional filters.`)
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
		fmt.Fprintln(c.out, "Usage:\n  crux agents [ls]")
	case "add":
		fmt.Fprintln(c.out, "Usage:\n  crux agents add <name> --cmd PATH [--arg ARG] [--env KEY=VALUE] [--description TEXT] [--workdir DIR] [--timeout SECONDS]")
	case "rm":
		fmt.Fprintln(c.out, "Usage:\n  crux agents rm <name>")
	case "describe":
		fmt.Fprintln(c.out, "Usage:\n  crux agents describe <name>")
	case "usage":
		fmt.Fprintln(c.out, "Usage:\n  crux agents usage")
	case "cost":
		fmt.Fprintln(c.out, "Usage:\n  crux agents cost")
	case "sessions":
		fmt.Fprintln(c.out, "Usage:\n  crux agents sessions")
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
	output, err := normalizeOutput(opts.output)
	if err != nil {
		return opts, "", nil, err
	}
	opts.output = output
	remaining := fs.Args()
	if len(remaining) == 0 {
		return opts, "", nil, nil
	}
	return opts, remaining[0], remaining[1:], nil
}

func applyOutputFlag(current string, args []string, output *string) ([]string, error) {
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-o" || arg == "--output":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%s requires a value: table, json, or yaml", arg)
			}
			value, err := normalizeOutput(args[i+1])
			if err != nil {
				return nil, err
			}
			*output = value
			i++
		case strings.HasPrefix(arg, "--output="):
			value, err := normalizeOutput(strings.TrimPrefix(arg, "--output="))
			if err != nil {
				return nil, err
			}
			*output = value
		case strings.HasPrefix(arg, "-o="):
			value, err := normalizeOutput(strings.TrimPrefix(arg, "-o="))
			if err != nil {
				return nil, err
			}
			*output = value
		default:
			filtered = append(filtered, arg)
		}
	}
	if *output == "" {
		*output = current
	}
	return filtered, nil
}

func normalizeOutput(format string) (string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		return "table", nil
	}
	switch format {
	case "table", "json", "yaml":
		return format, nil
	default:
		return "", fmt.Errorf("unsupported output format %q; expected table, json, or yaml", format)
	}
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
	installerOut := c.out
	if opts.output != "table" {
		installerOut = c.err
	}
	switch component {
	case "all":
		if opts.output == "table" {
			fmt.Fprintln(c.out, "updating crux and cruxd")
		}
		if err := c.runInstaller(ctx, cruxScriptURL, cruxPowerShellURL, installerArgs, env, installerOut); err != nil {
			return err
		}
	case "crux":
		if opts.output == "table" {
			fmt.Fprintln(c.out, "updating crux")
		}
		args := append([]string{}, installerArgs...)
		args = append(args, "--skip-cruxd")
		if err := c.runInstaller(ctx, cruxScriptURL, cruxPowerShellURL, args, env, installerOut); err != nil {
			return err
		}
	case "cruxd":
		if opts.output == "table" {
			fmt.Fprintln(c.out, "updating cruxd")
		}
		args := []string{"--version", cruxdVersion}
		if noStart {
			args = append(args, "--no-start")
		}
		if force {
			args = append(args, "--force")
		}
		if err := c.runInstaller(ctx, cruxdScriptURL, cruxdPowerShellURL, args, env, installerOut); err != nil {
			return err
		}
	}
	if noStart || component == "crux" {
		if opts.output != "table" {
			return c.print(opts.output, updateResult(component, cruxVersion, cruxdVersion, noStart, "installed"))
		}
		return nil
	}
	cl, ctxCfg, err := c.client(opts)
	if err != nil {
		return err
	}
	if err := waitForHealth(ctx, cl, postInstallHealthTimeout); err != nil {
		return fmt.Errorf("cruxd was updated but did not become healthy at %s: %w", ctxCfg.ServerURL, err)
	}
	if opts.output != "table" {
		result := updateResult(component, cruxVersion, cruxdVersion, noStart, "running")
		result.ServerURL = ctxCfg.ServerURL
		return c.print(opts.output, result)
	}
	fmt.Fprintf(c.out, "cruxd: running at %s\n", ctxCfg.ServerURL)
	return nil
}

func updateResult(component, cruxVersion, cruxdVersion string, noStart bool, status string) struct {
	Component    string `json:"component" yaml:"component"`
	CruxVersion  string `json:"cruxVersion,omitempty" yaml:"cruxVersion,omitempty"`
	CruxdVersion string `json:"cruxdVersion,omitempty" yaml:"cruxdVersion,omitempty"`
	Status       string `json:"status" yaml:"status"`
	ServerURL    string `json:"serverUrl,omitempty" yaml:"serverUrl,omitempty"`
	Started      bool   `json:"started" yaml:"started"`
} {
	return struct {
		Component    string `json:"component" yaml:"component"`
		CruxVersion  string `json:"cruxVersion,omitempty" yaml:"cruxVersion,omitempty"`
		CruxdVersion string `json:"cruxdVersion,omitempty" yaml:"cruxdVersion,omitempty"`
		Status       string `json:"status" yaml:"status"`
		ServerURL    string `json:"serverUrl,omitempty" yaml:"serverUrl,omitempty"`
		Started      bool   `json:"started" yaml:"started"`
	}{
		Component:    component,
		CruxVersion:  cruxVersion,
		CruxdVersion: cruxdVersion,
		Status:       status,
		Started:      !noStart && component != "crux",
	}
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
	if opts.output != "table" {
		return c.print(opts.output, struct {
			Daemon        string `json:"daemon" yaml:"daemon"`
			Status        string `json:"status" yaml:"status"`
			ServerVersion string `json:"serverVersion" yaml:"serverVersion"`
		}{
			Daemon:        "cruxd",
			Status:        "ok",
			ServerVersion: version,
		})
	}
	fmt.Fprintf(c.out, "cruxd: ok (%s)\n", version)
	return nil
}

func (c *CLI) version(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: crux version")
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	version, err := cl.Version(ctx)
	if err != nil {
		if opts.output != "table" {
			return c.print(opts.output, struct {
				Client          string `json:"client" yaml:"client"`
				Server          string `json:"server,omitempty" yaml:"server,omitempty"`
				ServerAvailable bool   `json:"serverAvailable" yaml:"serverAvailable"`
				Error           string `json:"error,omitempty" yaml:"error,omitempty"`
			}{
				Client:          cruxapi.Version,
				ServerAvailable: false,
				Error:           err.Error(),
			})
		}
		fmt.Fprintf(c.out, "crux client: %s\n", cruxapi.Version)
		fmt.Fprintf(c.out, "crux server: unavailable (%v)\n", err)
		return nil
	}
	if opts.output != "table" {
		return c.print(opts.output, struct {
			Client          string `json:"client" yaml:"client"`
			Server          string `json:"server" yaml:"server"`
			ServerAvailable bool   `json:"serverAvailable" yaml:"serverAvailable"`
		}{
			Client:          cruxapi.Version,
			Server:          version,
			ServerAvailable: true,
		})
	}
	fmt.Fprintf(c.out, "crux client: %s\n", cruxapi.Version)
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
		if opts.output != "table" {
			return c.print(opts.output, rows)
		}
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
		if opts.output != "table" {
			ctx, ok := cfg.Contexts[cfg.CurrentContext]
			if !ok {
				return fmt.Errorf("current context %q not found", cfg.CurrentContext)
			}
			return c.print(opts.output, struct {
				Name      string `json:"name" yaml:"name"`
				ServerURL string `json:"serverUrl" yaml:"serverUrl"`
				Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
			}{Name: cfg.CurrentContext, ServerURL: ctx.ServerURL, Namespace: ctx.Namespace})
		}
		fmt.Fprintln(c.out, cfg.CurrentContext)
	case "use":
		if len(args) != 2 {
			return fmt.Errorf("usage: crux context use <name>")
		}
		ctx, ok := cfg.Contexts[args[1]]
		if !ok {
			return fmt.Errorf("context %q not found", args[1])
		}
		cfg.CurrentContext = args[1]
		if err := config.SaveCLIConfig(path, cfg); err != nil {
			return err
		}
		if opts.output != "table" {
			return c.print(opts.output, struct {
				Name      string `json:"name" yaml:"name"`
				ServerURL string `json:"serverUrl" yaml:"serverUrl"`
				Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
				Current   bool   `json:"current" yaml:"current"`
			}{Name: args[1], ServerURL: ctx.ServerURL, Namespace: ctx.Namespace, Current: true})
		}
		fmt.Fprintf(c.out, "current context: %s\n", args[1])
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
		if err := config.SaveCLIConfig(path, cfg); err != nil {
			return err
		}
		if opts.output != "table" {
			return c.print(opts.output, struct {
				Name      string `json:"name" yaml:"name"`
				ServerURL string `json:"serverUrl" yaml:"serverUrl"`
				Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
				Current   bool   `json:"current" yaml:"current"`
			}{Name: args[1], ServerURL: serverURL, Namespace: namespace, Current: cfg.CurrentContext == args[1]})
		}
		fmt.Fprintf(c.out, "context %q set\n", args[1])
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
		if opts.output != "table" {
			return c.print(opts.output, runtime)
		}
		printRuntimeConfigTable(c.out, runtime)
		return nil
	case "set":
		patch, err := parseRuntimePatch(args[1:], c.err)
		if err != nil {
			return err
		}
		runtime, err := cl.UpdateRuntimeConfig(ctx, patch)
		if err != nil {
			return err
		}
		if opts.output != "table" {
			return c.print(opts.output, runtime)
		}
		printRuntimeConfigTable(c.out, runtime)
		return nil
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
		args = []string{"ls"}
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
		return c.describeAgent(ctx, opts, cl, args[1])
	case "add":
		return c.addAgent(ctx, opts, cl, args[1:])
	case "rm":
		if len(args) != 2 {
			return fmt.Errorf("usage: crux agents rm <name>")
		}
		return c.deleteAgent(ctx, opts, cl, args[1])
	case "usage":
		if len(args) != 1 {
			return fmt.Errorf("usage: crux agents usage")
		}
		return c.printFleetUsage(ctx, opts, cl)
	case "cost":
		if len(args) != 1 {
			return fmt.Errorf("usage: crux agents cost")
		}
		return c.printFleetCost(ctx, opts, cl)
	case "sessions":
		if len(args) != 1 {
			return fmt.Errorf("usage: crux agents sessions")
		}
		return c.printFleetSessions(ctx, opts, cl)
	default:
		return fmt.Errorf("unknown agents command %q", args[0])
	}
	return nil
}

func (c *CLI) agent(ctx context.Context, opts rootOptions, args []string) error {
	if len(args) == 0 || helpArg(args) {
		c.commandUsage("agent", nil)
		return nil
	}
	if len(args) >= 2 && (args[1] == "--help" || args[1] == "-h" || args[1] == "help") {
		c.commandUsage("agent", args[:1])
		return nil
	}
	if len(args) == 3 && (args[2] == "--help" || args[2] == "-h" || args[2] == "help") {
		c.commandUsage("agent", args[:2])
		return nil
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	action := "describe"
	if len(args) >= 2 {
		action = args[1]
	}
	switch action {
	case "describe":
		if len(args) > 2 {
			return fmt.Errorf("usage: crux agent <name> describe")
		}
		return c.describeAgent(ctx, opts, cl, args[0])
	case "usage":
		if len(args) > 2 {
			return fmt.Errorf("usage: crux agent <name> usage")
		}
		usage, err := cl.AgentUsage(ctx, args[0])
		if err != nil {
			return agentLookupError(args[0], err)
		}
		if opts.output != "table" {
			return c.print(opts.output, usage)
		}
		return c.printAgentUsage(usage)
	case "cost":
		if len(args) > 2 {
			return fmt.Errorf("usage: crux agent <name> cost")
		}
		cost, err := cl.AgentCost(ctx, args[0])
		if err != nil {
			return agentLookupError(args[0], err)
		}
		if opts.output != "table" {
			return c.print(opts.output, cost)
		}
		printCostTable(c.out, []cruxapi.AgentCostSnapshot{cost})
	case "sessions":
		if len(args) > 2 {
			return fmt.Errorf("usage: crux agent <name> sessions")
		}
		sessions, err := cl.AgentSessions(ctx, args[0])
		if err != nil {
			return agentLookupError(args[0], err)
		}
		if opts.output != "table" {
			return c.print(opts.output, sessions)
		}
		printSessionsTable(c.out, sessions)
	case "exec":
		return c.agentExec(ctx, opts, cl, args[0], args[2:])
	case "history":
		return c.agentHistory(ctx, opts, cl, args[0], args[2:])
	case "resume":
		return c.agentResume(ctx, opts, args[0], args[2:])
	case "fallback":
		return c.agentFallback(ctx, opts, cl, args[0], args[2:])
	case "rm":
		if len(args) > 2 {
			return fmt.Errorf("usage: crux agent <name> rm")
		}
		return c.deleteAgent(ctx, opts, cl, args[0])
	default:
		return fmt.Errorf("unknown agent command %q; expected describe, usage, cost, sessions, exec, history, resume, fallback, or rm", action)
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
	if opts.output != "table" {
		return c.print(opts.output, saved)
	}
	printAgentDetailTable(c.out, saved)
	return nil
}

func (c *CLI) describeAgent(ctx context.Context, opts rootOptions, cl *client.Client, name string) error {
	agent, err := findAgent(ctx, cl, name)
	if err != nil {
		return err
	}
	if opts.output != "table" {
		return c.print(opts.output, agent)
	}
	printAgentDetailTable(c.out, agent)
	return nil
}

func (c *CLI) deleteAgent(ctx context.Context, opts rootOptions, cl *client.Client, name string) error {
	cleanName := cruxapi.CleanAgentName(name)
	if err := cl.DeleteAgent(ctx, cleanName); err != nil {
		return agentLookupError(cleanName, err)
	}
	result := struct {
		Deleted   bool   `json:"deleted" yaml:"deleted"`
		AgentName string `json:"agentName" yaml:"agentName"`
	}{
		Deleted:   true,
		AgentName: cleanName,
	}
	if opts.output != "table" {
		return c.print(opts.output, result)
	}
	fmt.Fprintf(c.out, "agent %q removed\n", cleanName)
	return nil
}

func (c *CLI) printFleetUsage(ctx context.Context, opts rootOptions, cl *client.Client) error {
	agents, err := cl.ListAgents(ctx)
	if err != nil {
		return err
	}
	rows := make([]cruxapi.AgentUsage, 0, len(agents))
	for _, agent := range agents {
		usage, err := cl.AgentUsage(ctx, agent.Name)
		if err != nil {
			return err
		}
		rows = append(rows, usage)
	}
	if opts.output != "table" {
		return c.print(opts.output, rows)
	}
	fmt.Fprintf(c.out, "%-18s %-8s %-8s %-8s %-10s %-10s %-10s\n", "AGENT", "TOTAL", "OK", "FAILED", "RUNNING", "SUCCESS", "OUTPUT")
	for _, usage := range rows {
		fmt.Fprintf(c.out, "%-18s %-8d %-8d %-8d %-10d %-10.1f %-10d\n",
			usage.AgentName, usage.ExecutionsTotal, usage.Succeeded, usage.Failed, usage.Running, usage.SuccessRate*100, usage.StdoutBytes+usage.StderrBytes)
	}
	return nil
}

func (c *CLI) printFleetCost(ctx context.Context, opts rootOptions, cl *client.Client) error {
	agents, err := cl.ListAgents(ctx)
	if err != nil {
		return err
	}
	rows := make([]cruxapi.AgentCostSnapshot, 0, len(agents))
	for _, agent := range agents {
		cost, err := cl.AgentCost(ctx, agent.Name)
		if err != nil {
			return err
		}
		rows = append(rows, cost)
	}
	if opts.output != "table" {
		return c.print(opts.output, rows)
	}
	printCostTable(c.out, rows)
	return nil
}

func (c *CLI) printFleetSessions(ctx context.Context, opts rootOptions, cl *client.Client) error {
	agents, err := cl.ListAgents(ctx)
	if err != nil {
		return err
	}
	rows := make([]cruxapi.AgentSession, 0)
	for _, agent := range agents {
		sessions, err := cl.AgentSessions(ctx, agent.Name)
		if err != nil {
			return err
		}
		rows = append(rows, sessions...)
	}
	if opts.output != "table" {
		return c.print(opts.output, rows)
	}
	printSessionsTable(c.out, rows)
	return nil
}

func (c *CLI) agentHistory(ctx context.Context, opts rootOptions, cl *client.Client, name string, args []string) error {
	if len(args) == 0 {
		args = []string{"ls"}
	}
	switch args[0] {
	case "ls":
		if len(args) != 1 {
			return fmt.Errorf("usage: crux agent <name> history [ls]")
		}
		history, err := cl.AgentHistory(ctx, name)
		if err != nil {
			return agentLookupError(name, err)
		}
		if opts.output != "table" {
			return c.print(opts.output, history)
		}
		printHistoryTable(c.out, history)
	case "show":
		if len(args) != 2 {
			return fmt.Errorf("usage: crux agent <name> history show <execution-id>")
		}
		execution, err := cl.GetExecution(ctx, args[1])
		if err != nil {
			return err
		}
		if execution.AgentName != cruxapi.CleanAgentName(name) {
			return fmt.Errorf("execution %s belongs to agent %q", execution.ID, execution.AgentName)
		}
		if opts.output != "table" {
			return c.print(opts.output, execution)
		}
		printExecutionDetail(c.out, execution)
	case "share":
		if len(args) < 3 {
			return fmt.Errorf("usage: crux agent <name> history share <execution-id> <target-agent> [--prompt TEXT]")
		}
		runOpts := runOptions{
			AgentName:    args[2],
			SourceExecID: args[1],
		}
		for i := 3; i < len(args); i++ {
			switch {
			case args[i] == "--prompt" || args[i] == "--edit":
				i++
				if i >= len(args) {
					return fmt.Errorf("%s requires text", args[i-1])
				}
				runOpts.Prompt = args[i]
			case strings.HasPrefix(args[i], "--prompt="):
				runOpts.Prompt = strings.TrimPrefix(args[i], "--prompt=")
			case strings.HasPrefix(args[i], "--edit="):
				runOpts.Prompt = strings.TrimPrefix(args[i], "--edit=")
			default:
				return fmt.Errorf("usage: crux agent <name> history share <execution-id> <target-agent> [--prompt TEXT]")
			}
		}
		return c.executeRun(ctx, opts, cl, runOpts)
	default:
		return fmt.Errorf("unknown history command %q; expected ls, show, or share", args[0])
	}
	return nil
}

type agentExecOptions struct {
	ResumeSession  string
	WorkingDir     string
	Driver         string
	TimeoutSeconds int
	TranscriptPath string
	Send           []string
	Expect         []string
	ProviderArgs   []string
	DryRun         bool
	NoRecord       bool
}

func (c *CLI) agentExec(ctx context.Context, opts rootOptions, cl *client.Client, name string, args []string) error {
	execOpts, err := parseAgentExecArgs(args)
	if err != nil {
		return err
	}
	if execOpts.WorkingDir == "" {
		execOpts.WorkingDir = currentWorkingDir()
	}
	plan, err := cl.AgentExecPlan(ctx, name, cruxapi.AgentExecPlanRequest{
		WorkingDir:    execOpts.WorkingDir,
		ResumeSession: execOpts.ResumeSession,
		Args:          execOpts.ProviderArgs,
	})
	if err != nil {
		return agentLookupError(name, err)
	}
	if execOpts.DryRun {
		if opts.output != "table" {
			return c.print(opts.output, plan)
		}
		printExecPlan(c.out, plan)
		return nil
	}
	if execOpts.TranscriptPath == "" {
		execOpts.TranscriptPath, err = defaultTTYTranscriptPath(plan.AgentName)
		if err != nil {
			return err
		}
	}
	started := time.Now().UTC()
	result := c.runTTYCommand(plan.Command, execOpts)
	completed := time.Now().UTC()
	transcript := ""
	if execOpts.TranscriptPath != "" {
		data, readErr := os.ReadFile(execOpts.TranscriptPath)
		if readErr == nil {
			transcript = string(data)
		} else if result.Error == "" {
			result.Error = readErr.Error()
		}
	}
	if missing := missingExpectations(transcript, execOpts.Expect); len(missing) > 0 {
		if result.Error != "" {
			result.Error += "; "
		}
		result.Error += "missing expected TTY output: " + strings.Join(missing, ", ")
		if result.ExitCode == 0 {
			result.ExitCode = 1
		}
	}
	if execOpts.NoRecord {
		if opts.output != "table" {
			return c.print(opts.output, struct {
				Plan           cruxapi.AgentExecPlan `json:"plan" yaml:"plan"`
				Driver         string                `json:"driver" yaml:"driver"`
				TranscriptPath string                `json:"transcriptPath" yaml:"transcriptPath"`
				ExitCode       int                   `json:"exitCode" yaml:"exitCode"`
				Error          string                `json:"error,omitempty" yaml:"error,omitempty"`
			}{Plan: plan, Driver: result.Driver, TranscriptPath: execOpts.TranscriptPath, ExitCode: result.ExitCode, Error: result.Error})
		}
		fmt.Fprintf(c.out, "TTY exec finished exit=%d driver=%s transcript=%s\n", result.ExitCode, result.Driver, execOpts.TranscriptPath)
		if result.Error != "" {
			return errors.New(result.Error)
		}
		return nil
	}
	record, err := cl.RecordAgentExec(ctx, plan.AgentName, cruxapi.AgentExecRecordRequest{
		WorkingDir:     plan.Command.WorkingDir,
		ResumeSession:  execOpts.ResumeSession,
		Args:           plan.Command.Args,
		Driver:         result.Driver,
		TranscriptPath: execOpts.TranscriptPath,
		Transcript:     transcript,
		Stderr:         result.Stderr,
		Error:          result.Error,
		ExitCode:       result.ExitCode,
		StartedAt:      started,
		CompletedAt:    completed,
	})
	if err != nil {
		return err
	}
	if opts.output != "table" {
		if err := c.print(opts.output, record); err != nil {
			return err
		}
		if record.Execution.Status != cruxapi.ExecutionSucceeded {
			return fmt.Errorf("execution %s failed: %s", record.Execution.ID, record.Execution.Error)
		}
		return nil
	}
	fmt.Fprintf(c.out, "Recorded TTY execution: %s status=%s exit=%d driver=%s\n", record.Execution.ID, record.Execution.Status, record.Execution.ExitCode, result.Driver)
	fmt.Fprintf(c.out, "Transcript: %s\n", execOpts.TranscriptPath)
	fmt.Fprintf(c.out, "Usage: total=%d succeeded=%d failed=%d running=%d outputBytes=%d\n",
		record.Usage.ExecutionsTotal, record.Usage.Succeeded, record.Usage.Failed, record.Usage.Running, record.Usage.StdoutBytes+record.Usage.StderrBytes)
	fmt.Fprintf(c.out, "Sessions: %d visible for %s\n", len(record.Sessions), record.Execution.AgentName)
	if record.Execution.Status != cruxapi.ExecutionSucceeded {
		return fmt.Errorf("execution %s failed: %s", record.Execution.ID, record.Execution.Error)
	}
	return nil
}

func parseAgentExecArgs(args []string) (agentExecOptions, error) {
	var out agentExecOptions
	var sendFlags, expectFlags multiFlag
	fs := flag.NewFlagSet("agent exec", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&out.ResumeSession, "resume", "", "provider session id or last")
	fs.StringVar(&out.WorkingDir, "workdir", "", "working directory")
	fs.StringVar(&out.Driver, "driver", "auto", "auto, script, expect, or direct")
	fs.IntVar(&out.TimeoutSeconds, "timeout", 0, "timeout seconds for scripted input mode")
	fs.StringVar(&out.TranscriptPath, "transcript", "", "transcript path")
	fs.BoolVar(&out.DryRun, "dry-run", false, "print the provider TTY command without running it")
	fs.BoolVar(&out.NoRecord, "no-record", false, "do not import the transcript into Crux history")
	fs.Var(&sendFlags, "send", "input text to send to the TTY; repeatable")
	fs.Var(&sendFlags, "input", "alias for --send")
	fs.Var(&expectFlags, "expect", "text expected in the TTY transcript; repeatable")
	if err := fs.Parse(args); err != nil {
		return out, err
	}
	out.Send = append([]string{}, sendFlags...)
	out.Expect = append([]string{}, expectFlags...)
	out.ProviderArgs = append([]string{}, fs.Args()...)
	switch out.Driver {
	case "auto", "script", "expect", "direct":
	default:
		return out, fmt.Errorf("--driver must be auto, script, expect, or direct")
	}
	if out.WorkingDir != "" && !filepath.IsAbs(out.WorkingDir) {
		return out, fmt.Errorf("--workdir must be an absolute path")
	}
	return out, nil
}

func (c *CLI) agentResume(ctx context.Context, opts rootOptions, name string, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: crux agent <name> resume <session-id|last> <prompt> [--async] [--fallback AGENT[,AGENT]]")
	}
	runOpts, err := parseRunArgs(append([]string{name, "--resume", args[0]}, args[1:]...))
	if err != nil {
		return err
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	return c.executeRun(ctx, opts, cl, runOpts)
}

func (c *CLI) agentFallback(ctx context.Context, opts rootOptions, cl *client.Client, name string, args []string) error {
	agent, err := findAgent(ctx, cl, name)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		fallbacks := fallbackAgents(agent)
		if opts.output != "table" {
			return c.print(opts.output, struct {
				AgentName string   `json:"agentName" yaml:"agentName"`
				Fallbacks []string `json:"fallbacks" yaml:"fallbacks"`
			}{AgentName: agent.Name, Fallbacks: fallbacks})
		}
		fmt.Fprintf(c.out, "Agent: %s\n", agent.Name)
		fmt.Fprintf(c.out, "Fallbacks: %s\n", firstNonEmpty(strings.Join(fallbacks, ","), "-"))
		return nil
	}
	switch args[0] {
	case "set":
		if len(args) != 2 {
			return fmt.Errorf("usage: crux agent <name> fallback set <agent>[,<agent>...]")
		}
		setFallbackAgents(&agent, splitCSV(args[1]))
	case "clear":
		if len(args) != 1 {
			return fmt.Errorf("usage: crux agent <name> fallback clear")
		}
		setFallbackAgents(&agent, nil)
	default:
		return fmt.Errorf("unknown fallback command %q; expected set or clear", args[0])
	}
	saved, err := cl.UpsertAgent(ctx, agent)
	if err != nil {
		return err
	}
	if opts.output != "table" {
		return c.print(opts.output, saved)
	}
	fmt.Fprintf(c.out, "agent %q fallback set to %s\n", saved.Name, firstNonEmpty(strings.Join(fallbackAgents(saved), ","), "-"))
	return nil
}

func findAgent(ctx context.Context, cl *client.Client, name string) (cruxapi.Agent, error) {
	cleanName := cruxapi.CleanAgentName(name)
	agents, err := cl.ListAgents(ctx)
	if err != nil {
		return cruxapi.Agent{}, err
	}
	for _, agent := range agents {
		if agent.Name == cleanName {
			return agent, nil
		}
	}
	return cruxapi.Agent{}, fmt.Errorf("agent %q not found; run \"crux discover\" first or add it with \"crux agents add\"", name)
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
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	runOpts, err := parseRunArgs(args)
	if err != nil {
		return err
	}
	return c.executeRun(ctx, opts, cl, runOpts)
}

type runOptions struct {
	AgentName      string
	Prompt         string
	Async          bool
	ResumeSession  string
	SourceExecID   string
	FallbackAgents []string
}

func parseRunArgs(args []string) (runOptions, error) {
	var out runOptions
	positionals := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--async":
			out.Async = true
		case arg == "--fallback":
			i++
			if i >= len(args) {
				return out, fmt.Errorf("--fallback requires a value")
			}
			out.FallbackAgents = append(out.FallbackAgents, splitCSV(args[i])...)
		case strings.HasPrefix(arg, "--fallback="):
			out.FallbackAgents = append(out.FallbackAgents, splitCSV(strings.TrimPrefix(arg, "--fallback="))...)
		case arg == "--resume":
			i++
			if i >= len(args) {
				return out, fmt.Errorf("--resume requires a session id")
			}
			out.ResumeSession = args[i]
		case strings.HasPrefix(arg, "--resume="):
			out.ResumeSession = strings.TrimPrefix(arg, "--resume=")
		case arg == "--from":
			i++
			if i >= len(args) {
				return out, fmt.Errorf("--from requires an execution id")
			}
			out.SourceExecID = args[i]
		case strings.HasPrefix(arg, "--from="):
			out.SourceExecID = strings.TrimPrefix(arg, "--from=")
		case arg == "--prompt" || arg == "--edit":
			i++
			if i >= len(args) {
				return out, fmt.Errorf("%s requires text", arg)
			}
			out.Prompt = args[i]
		case strings.HasPrefix(arg, "--prompt="):
			out.Prompt = strings.TrimPrefix(arg, "--prompt=")
		case strings.HasPrefix(arg, "--edit="):
			out.Prompt = strings.TrimPrefix(arg, "--edit=")
		default:
			positionals = append(positionals, arg)
		}
	}
	if len(positionals) == 0 {
		return out, fmt.Errorf("usage: crux run <agent> <prompt> [--async] [--resume SESSION] [--fallback AGENT[,AGENT]]")
	}
	out.AgentName = positionals[0]
	if out.Prompt == "" && len(positionals) > 1 {
		out.Prompt = strings.Join(positionals[1:], " ")
	}
	if out.Prompt == "" && out.SourceExecID == "" {
		return out, fmt.Errorf("usage: crux run <agent> <prompt> [--async] [--resume SESSION] [--fallback AGENT[,AGENT]]")
	}
	out.FallbackAgents = uniqueAgentNames(out.FallbackAgents)
	return out, nil
}

type runAttempt struct {
	Agent     string            `json:"agent" yaml:"agent"`
	Execution cruxapi.Execution `json:"execution" yaml:"execution"`
	Error     string            `json:"error,omitempty" yaml:"error,omitempty"`
}

type runChainResult struct {
	Attempts []runAttempt      `json:"attempts" yaml:"attempts"`
	Final    cruxapi.Execution `json:"final" yaml:"final"`
}

func (c *CLI) executeRun(ctx context.Context, opts rootOptions, cl *client.Client, runOpts runOptions) error {
	if runOpts.SourceExecID != "" {
		prompt, err := c.promptFromHistory(ctx, cl, runOpts.SourceExecID, runOpts.Prompt)
		if err != nil {
			return err
		}
		runOpts.Prompt = prompt
	}
	if len(runOpts.FallbackAgents) == 0 {
		agent, err := findAgent(ctx, cl, runOpts.AgentName)
		if err == nil {
			runOpts.FallbackAgents = fallbackAgents(agent)
		}
	}
	chain := []string{cruxapi.CleanAgentName(runOpts.AgentName)}
	chain = append(chain, runOpts.FallbackAgents...)
	chain = uniqueAgentNames(chain)
	attempts := make([]runAttempt, 0, len(chain))
	var final cruxapi.Execution
	var lastErr error
	for _, agentName := range chain {
		execution, err := cl.Run(ctx, cruxapi.SubmitExecutionRequest{
			AgentName:      agentName,
			Prompt:         runOpts.Prompt,
			WorkingDir:     currentWorkingDir(),
			ResumeSession:  runOpts.ResumeSession,
			SourceExecID:   runOpts.SourceExecID,
			FallbackAgents: runOpts.FallbackAgents,
			Wait:           !runOpts.Async,
		})
		attempt := runAttempt{Agent: agentName, Execution: execution}
		if err != nil {
			attempt.Error = err.Error()
			attempts = append(attempts, attempt)
			lastErr = agentLookupError(agentName, err)
			break
		}
		attempts = append(attempts, attempt)
		final = execution
		if runOpts.Async || execution.Status == cruxapi.ExecutionSucceeded {
			break
		}
		lastErr = fmt.Errorf("execution %s failed: %s", execution.ID, execution.Error)
	}
	result := runChainResult{Attempts: attempts, Final: final}
	if opts.output != "table" {
		if err := c.print(opts.output, result); err != nil {
			return err
		}
		if !runOpts.Async && final.Status != cruxapi.ExecutionSucceeded {
			if lastErr != nil {
				return lastErr
			}
			return fmt.Errorf("execution failed")
		}
		return nil
	}
	if runOpts.Async {
		executions := make([]cruxapi.Execution, 0, len(attempts))
		for _, attempt := range attempts {
			executions = append(executions, attempt.Execution)
		}
		printExecutionTable(c.out, executions)
		return nil
	}
	if final.Stdout != "" {
		fmt.Fprint(c.out, final.Stdout)
	}
	if final.Stderr != "" {
		fmt.Fprint(c.err, final.Stderr)
	}
	if final.Status != cruxapi.ExecutionSucceeded {
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("execution %s failed: %s", final.ID, final.Error)
	}
	if len(attempts) > 1 {
		fmt.Fprintf(c.err, "crux fallback: succeeded with %s after %d attempts\n", final.AgentName, len(attempts))
	}
	return nil
}

func (c *CLI) promptFromHistory(ctx context.Context, cl *client.Client, executionID, override string) (string, error) {
	source, err := cl.GetExecution(ctx, executionID)
	if err != nil {
		return "", err
	}
	task := strings.TrimSpace(override)
	if task == "" {
		task = source.Prompt
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "Crux shared execution context from %s (%s).\n\n", source.ID, source.AgentName)
	if strings.TrimSpace(source.Prompt) != "" {
		fmt.Fprintf(&builder, "Original prompt:\n%s\n\n", source.Prompt)
	}
	if strings.TrimSpace(source.Stdout) != "" {
		fmt.Fprintf(&builder, "Prior stdout:\n%s\n\n", source.Stdout)
	}
	if strings.TrimSpace(source.Stderr) != "" {
		fmt.Fprintf(&builder, "Prior stderr:\n%s\n\n", source.Stderr)
	}
	if strings.TrimSpace(source.Error) != "" {
		fmt.Fprintf(&builder, "Prior error:\n%s\n\n", source.Error)
	}
	fmt.Fprintf(&builder, "New task:\n%s", task)
	return builder.String(), nil
}

func (c *CLI) ps(ctx context.Context, opts rootOptions, args []string) error {
	filter, err := parsePSArgs(args)
	if err != nil {
		return err
	}
	cl, _, err := c.client(opts)
	if err != nil {
		return err
	}
	executions, err := cl.ListExecutions(ctx)
	if err != nil {
		return err
	}
	executions = filterExecutions(executions, filter)
	if opts.output != "table" {
		return c.print(opts.output, executions)
	}
	printExecutionTable(c.out, executions)
	return nil
}

type psFilter struct {
	Agent  string
	Status string
	Last   int
}

func parsePSArgs(args []string) (psFilter, error) {
	var filter psFilter
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--agent":
			i++
			if i >= len(args) {
				return filter, fmt.Errorf("--agent requires a name")
			}
			filter.Agent = cruxapi.CleanAgentName(args[i])
		case strings.HasPrefix(arg, "--agent="):
			filter.Agent = cruxapi.CleanAgentName(strings.TrimPrefix(arg, "--agent="))
		case arg == "--status":
			i++
			if i >= len(args) {
				return filter, fmt.Errorf("--status requires a value")
			}
			filter.Status = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--status="):
			filter.Status = strings.TrimSpace(strings.TrimPrefix(arg, "--status="))
		case arg == "--last":
			i++
			if i >= len(args) {
				return filter, fmt.Errorf("--last requires a number")
			}
			value, err := strconv.Atoi(args[i])
			if err != nil || value < 1 {
				return filter, fmt.Errorf("--last must be a positive integer")
			}
			filter.Last = value
		case strings.HasPrefix(arg, "--last="):
			value, err := strconv.Atoi(strings.TrimPrefix(arg, "--last="))
			if err != nil || value < 1 {
				return filter, fmt.Errorf("--last must be a positive integer")
			}
			filter.Last = value
		default:
			return filter, fmt.Errorf("usage: crux ps [--agent NAME] [--status STATUS] [--last N]")
		}
	}
	return filter, nil
}

func filterExecutions(executions []cruxapi.Execution, filter psFilter) []cruxapi.Execution {
	out := make([]cruxapi.Execution, 0, len(executions))
	for _, execution := range executions {
		if filter.Agent != "" && execution.AgentName != filter.Agent {
			continue
		}
		if filter.Status != "" && string(execution.Status) != filter.Status {
			continue
		}
		out = append(out, execution)
		if filter.Last > 0 && len(out) >= filter.Last {
			break
		}
	}
	return out
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
		encoder := json.NewEncoder(c.out)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
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
	if usage.Description != "" || usage.Status != "" || usage.Version != "" {
		fmt.Fprintf(c.out, "Metadata: status=%s version=%s description=%s\n",
			firstNonEmpty(string(usage.Status), "-"), firstNonEmpty(usage.Version, "-"), firstNonEmpty(usage.Description, "-"))
	}
	if usage.CommandPath != "" {
		fmt.Fprintf(c.out, "Command: %s %s\n", usage.CommandPath, strings.Join(usage.CommandArgs, " "))
	}
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
		if usage.LastStdout != "" {
			fmt.Fprintf(c.out, "LastStdout: %s\n", oneLine(usage.LastStdout))
		}
		if usage.LastStderr != "" {
			fmt.Fprintf(c.out, "LastStderr: %s\n", oneLine(usage.LastStderr))
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
		fmt.Fprintln(c.out, "Live probes:")
		for _, metric := range usage.ExternalMetrics {
			status := "not available"
			if metric.Available {
				status = firstNonEmpty(metric.Value, "available")
			} else if metric.Value != "" {
				status = "not available (" + metric.Value + ")"
			}
			fmt.Fprintf(c.out, "  %s: %s", metric.Name, oneLine(status))
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

func printCostTable(out io.Writer, rows []cruxapi.AgentCostSnapshot) {
	fmt.Fprintf(out, "%-18s %-8s %-8s %-8s %-10s %-10s %-10s %s\n", "AGENT", "PROVIDER", "TOTAL", "RUNNING", "SUCCESS", "DURATION", "OUTPUT", "PROVIDER")
	for _, row := range rows {
		provider := "unavailable"
		if row.ProviderCostAvailable {
			provider = oneLine(row.ProviderCostValue)
		} else if row.ProviderCostDescription != "" {
			provider = oneLine(row.ProviderCostDescription)
		}
		fmt.Fprintf(out, "%-18s %-8s %-8d %-8d %-10.1f %-10s %-10d %s\n",
			row.AgentName, row.Provider, row.ExecutionsTotal, row.Running, row.SuccessRate*100,
			formatSeconds(row.TotalDurationSeconds), row.StdoutBytes+row.StderrBytes, provider)
	}
}

func printSessionsTable(out io.Writer, rows []cruxapi.AgentSession) {
	fmt.Fprintf(out, "%-18s %-10s %-10s %-36s %-8s %-36s %s\n", "AGENT", "PROVIDER", "SOURCE", "ID", "RESUME", "WORKDIR", "TITLE")
	for _, row := range rows {
		resume := "no"
		if row.ResumeSupported {
			resume = "yes"
		}
		title := row.Title
		if row.Age != "" {
			title = title + " (" + row.Age + ")"
		}
		fmt.Fprintf(out, "%-18s %-10s %-10s %-36s %-8s %-36s %s\n",
			row.AgentName, row.Provider, row.Source, row.ID, resume, compactPath(row.WorkingDir), oneLine(title))
	}
}

func printHistoryTable(out io.Writer, rows []cruxapi.AgentHistoryItem) {
	fmt.Fprintf(out, "%-28s %-18s %-10s %-5s %-20s %s\n", "ID", "AGENT", "STATUS", "EXIT", "QUEUED", "PROMPT")
	for _, row := range rows {
		fmt.Fprintf(out, "%-28s %-18s %-10s %-5d %-20s %s\n",
			row.ID, row.AgentName, row.Status, row.ExitCode, row.QueuedAt.Format(time.RFC3339), oneLine(row.PromptPreview))
	}
}

func printExecutionDetail(out io.Writer, execution cruxapi.Execution) {
	fmt.Fprintf(out, "ID: %s\n", execution.ID)
	fmt.Fprintf(out, "Agent: %s\n", execution.AgentName)
	fmt.Fprintf(out, "Status: %s\n", execution.Status)
	fmt.Fprintf(out, "Exit: %d\n", execution.ExitCode)
	if execution.WorkingDir != "" {
		fmt.Fprintf(out, "WorkingDir: %s\n", execution.WorkingDir)
	}
	if execution.ResumeSession != "" {
		fmt.Fprintf(out, "ResumeSession: %s\n", execution.ResumeSession)
	}
	if execution.SourceExecID != "" {
		fmt.Fprintf(out, "SourceExecution: %s\n", execution.SourceExecID)
	}
	fmt.Fprintf(out, "Queued: %s\n", execution.QueuedAt.Format(time.RFC3339))
	if execution.Error != "" {
		fmt.Fprintf(out, "Error: %s\n", execution.Error)
	}
	if execution.Prompt != "" {
		fmt.Fprintf(out, "\nPrompt:\n%s\n", execution.Prompt)
	}
	if execution.Stdout != "" {
		fmt.Fprintf(out, "\nStdout:\n%s\n", execution.Stdout)
	}
	if execution.Stderr != "" {
		fmt.Fprintf(out, "\nStderr:\n%s\n", execution.Stderr)
	}
}

func printAgentDetailTable(out io.Writer, agent cruxapi.Agent) {
	fmt.Fprintf(out, "Name: %s\n", agent.Name)
	fmt.Fprintf(out, "ID: %s\n", firstNonEmpty(agent.ID, "-"))
	fmt.Fprintf(out, "Status: %s\n", firstNonEmpty(string(agent.Status), "-"))
	if agent.Description != "" {
		fmt.Fprintf(out, "Description: %s\n", agent.Description)
	}
	fmt.Fprintf(out, "Command: %s %s\n", agent.Command.Path, strings.Join(agent.Command.Args, " "))
	if agent.Command.WorkingDir != "" {
		fmt.Fprintf(out, "WorkingDir: %s\n", agent.Command.WorkingDir)
	}
	if agent.Command.TimeoutSeconds > 0 {
		fmt.Fprintf(out, "Timeout: %ds\n", agent.Command.TimeoutSeconds)
	}
	if len(agent.Labels) > 0 {
		fmt.Fprintln(out, "Labels:")
		keys := make([]string, 0, len(agent.Labels))
		for key := range agent.Labels {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(out, "  %s: %s\n", key, agent.Labels[key])
		}
	}
	if !agent.CreatedAt.IsZero() {
		fmt.Fprintf(out, "Created: %s\n", agent.CreatedAt.Format(time.RFC3339))
	}
	if !agent.UpdatedAt.IsZero() {
		fmt.Fprintf(out, "Updated: %s\n", agent.UpdatedAt.Format(time.RFC3339))
	}
}

func printRuntimeConfigTable(out io.Writer, runtime cruxapi.RuntimeConfig) {
	fmt.Fprintf(out, "%-28s %s\n", "KEY", "VALUE")
	fmt.Fprintf(out, "%-28s %d\n", "workerConcurrency", runtime.WorkerConcurrency)
	fmt.Fprintf(out, "%-28s %d\n", "jobTimeoutSeconds", runtime.JobTimeoutSeconds)
	fmt.Fprintf(out, "%-28s %d\n", "maxOutputBytes", runtime.MaxOutputBytes)
	fmt.Fprintf(out, "%-28s %d\n", "discoveryTimeoutSeconds", runtime.DiscoveryTimeoutSecs)
	fmt.Fprintf(out, "%-28s %s\n", "logLevel", runtime.LogLevel)
	fmt.Fprintf(out, "%-28s %s\n", "defaultNamespace", runtime.DefaultNamespace)
	fmt.Fprintf(out, "%-28s %t\n", "allowShellCommands", runtime.AllowShellCommands)
	fmt.Fprintf(out, "%-28s %d\n", "traceRetentionEntries", runtime.TraceRetentionEntries)
}

func printExecutionTable(out io.Writer, executions []cruxapi.Execution) {
	fmt.Fprintf(out, "%-29s %-18s %-10s %-10s %-6s %s\n", "ID", "AGENT", "STATUS", "DURATION", "EXIT", "QUEUED")
	for _, execution := range executions {
		fmt.Fprintf(out, "%-29s %-18s %-10s %-10s %-6d %s\n", execution.ID, execution.AgentName, execution.Status, executionDuration(execution), execution.ExitCode, execution.QueuedAt.Format(time.RFC3339))
	}
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

func agentLookupError(name string, err error) error {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
		return fmt.Errorf("agent %q not found; run \"crux discover\" first or add it with \"crux agents add\"", name)
	}
	return err
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

func currentWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

const fallbackAgentsLabel = "cruxctl.io/fallback-agents"

func fallbackAgents(agent cruxapi.Agent) []string {
	if agent.Labels == nil {
		return nil
	}
	return uniqueAgentNames(splitCSV(agent.Labels[fallbackAgentsLabel]))
}

func setFallbackAgents(agent *cruxapi.Agent, fallbacks []string) {
	if agent.Labels == nil {
		agent.Labels = map[string]string{}
	}
	values := uniqueAgentNames(fallbacks)
	if len(values) == 0 {
		delete(agent.Labels, fallbackAgentsLabel)
		return
	}
	agent.Labels[fallbackAgentsLabel] = strings.Join(values, ",")
}

func splitCSV(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' '
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		name := cruxapi.CleanAgentName(part)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

func uniqueAgentNames(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		name := cruxapi.CleanAgentName(value)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func oneLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " | ")
	if len(value) > 240 {
		return value[:240] + "...(truncated)"
	}
	return value
}

func compactPath(value string) string {
	value = oneLine(value)
	if value == "" {
		return "-"
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if strings.HasPrefix(value, home) {
			value = "~" + strings.TrimPrefix(value, home)
		}
	}
	if len(value) <= 36 {
		return value
	}
	return "..." + value[len(value)-33:]
}

type ttyExecResult struct {
	Driver   string
	ExitCode int
	Stderr   string
	Error    string
}

func (c *CLI) runTTYCommand(command cruxapi.CommandSpec, opts agentExecOptions) ttyExecResult {
	driver := selectTTYDriver(opts.Driver)
	switch driver {
	case "script":
		return c.runScriptTTY(command, opts)
	case "expect":
		return c.runExpectTTY(command, opts)
	default:
		return c.runDirectTTY(command, opts, driver)
	}
}

func selectTTYDriver(requested string) string {
	switch requested {
	case "script":
		return "script"
	case "expect":
		if _, err := exec.LookPath("expect"); err == nil {
			return "expect"
		}
		return "expect"
	case "direct":
		return "direct"
	default:
		if _, err := exec.LookPath("expect"); err == nil {
			return "expect"
		}
		if scriptPath, err := exec.LookPath("script"); err == nil && scriptSupportsCommand(scriptPath) {
			return "script"
		}
		return "direct"
	}
}

func (c *CLI) runScriptTTY(command cruxapi.CommandSpec, opts agentExecOptions) ttyExecResult {
	scriptPath, err := exec.LookPath("script")
	if err != nil {
		return ttyExecResult{Driver: "script", ExitCode: 127, Error: "script command not found"}
	}
	if !scriptSupportsCommand(scriptPath) {
		return ttyExecResult{Driver: "script", ExitCode: 2, Error: "script command does not support command execution with -c"}
	}
	if err := os.MkdirAll(filepath.Dir(opts.TranscriptPath), 0o755); err != nil {
		return ttyExecResult{Driver: "script", ExitCode: 1, Error: err.Error()}
	}
	args := []string{"-q", "-e", "-f", "-c", shellCommand(command), opts.TranscriptPath}
	runCtx, cancel := ttyCommandContext(opts)
	defer cancel()
	cmd := exec.CommandContext(runCtx, scriptPath, args...)
	cmd.Env = commandEnv(command)
	if strings.TrimSpace(command.WorkingDir) != "" {
		cmd.Dir = command.WorkingDir
	}
	cmd.Stdout = c.out
	var stderr strings.Builder
	cmd.Stderr = io.MultiWriter(c.err, &stderr)
	if len(opts.Send) > 0 {
		cmd.Stdin = strings.NewReader(ttyInput(opts.Send))
	} else {
		cmd.Stdin = os.Stdin
	}
	err = cmd.Run()
	return applyTTYTimeout(ttyResult("script", err, stderr.String()), runCtx, opts)
}

func (c *CLI) runExpectTTY(command cruxapi.CommandSpec, opts agentExecOptions) ttyExecResult {
	expectPath, err := exec.LookPath("expect")
	if err != nil {
		return ttyExecResult{Driver: "expect", ExitCode: 127, Error: "expect command not found"}
	}
	if err := os.MkdirAll(filepath.Dir(opts.TranscriptPath), 0o755); err != nil {
		return ttyExecResult{Driver: "expect", ExitCode: 1, Error: err.Error()}
	}
	scriptFile, err := os.CreateTemp("", "crux-expect-*.exp")
	if err != nil {
		return ttyExecResult{Driver: "expect", ExitCode: 1, Error: err.Error()}
	}
	scriptName := scriptFile.Name()
	defer os.Remove(scriptName)
	_, writeErr := scriptFile.WriteString(expectProgram(opts))
	closeErr := scriptFile.Close()
	if writeErr != nil {
		return ttyExecResult{Driver: "expect", ExitCode: 1, Error: writeErr.Error()}
	}
	if closeErr != nil {
		return ttyExecResult{Driver: "expect", ExitCode: 1, Error: closeErr.Error()}
	}
	args := append([]string{scriptName, command.Path}, command.Args...)
	cmd := exec.Command(expectPath, args...)
	cmd.Env = append(commandEnv(command), "CRUX_TTY_TRANSCRIPT="+opts.TranscriptPath)
	if strings.TrimSpace(command.WorkingDir) != "" {
		cmd.Dir = command.WorkingDir
	}
	cmd.Stdout = c.out
	var stderr strings.Builder
	cmd.Stderr = io.MultiWriter(c.err, &stderr)
	cmd.Stdin = os.Stdin
	err = cmd.Run()
	return ttyResult("expect", err, stderr.String())
}

func (c *CLI) runDirectTTY(command cruxapi.CommandSpec, opts agentExecOptions, driver string) ttyExecResult {
	if opts.TranscriptPath != "" {
		if err := os.MkdirAll(filepath.Dir(opts.TranscriptPath), 0o755); err != nil {
			return ttyExecResult{Driver: driver, ExitCode: 1, Error: err.Error()}
		}
	}
	var transcript *os.File
	if opts.TranscriptPath != "" {
		var err error
		transcript, err = os.Create(opts.TranscriptPath)
		if err != nil {
			return ttyExecResult{Driver: driver, ExitCode: 1, Error: err.Error()}
		}
		defer transcript.Close()
	}
	runCtx, cancel := ttyCommandContext(opts)
	defer cancel()
	cmd := exec.CommandContext(runCtx, command.Path, command.Args...)
	cmd.Env = commandEnv(command)
	if strings.TrimSpace(command.WorkingDir) != "" {
		cmd.Dir = command.WorkingDir
	}
	stdout := io.Writer(c.out)
	if transcript != nil {
		stdout = io.MultiWriter(c.out, transcript)
	}
	cmd.Stdout = stdout
	var stderr strings.Builder
	stderrWriters := []io.Writer{c.err, &stderr}
	if transcript != nil {
		stderrWriters = append(stderrWriters, transcript)
	}
	cmd.Stderr = io.MultiWriter(stderrWriters...)
	if len(opts.Send) > 0 {
		cmd.Stdin = strings.NewReader(ttyInput(opts.Send))
	} else {
		cmd.Stdin = os.Stdin
	}
	err := cmd.Run()
	return applyTTYTimeout(ttyResult(driver, err, stderr.String()), runCtx, opts)
}

func scriptSupportsCommand(scriptPath string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, scriptPath, "--help").CombinedOutput()
	return err == nil && strings.Contains(string(out), "--command")
}

func ttyCommandContext(opts agentExecOptions) (context.Context, context.CancelFunc) {
	if opts.TimeoutSeconds > 0 && len(opts.Send) > 0 {
		return context.WithTimeout(context.Background(), time.Duration(opts.TimeoutSeconds)*time.Second)
	}
	return context.Background(), func() {}
}

func applyTTYTimeout(result ttyExecResult, ctx context.Context, opts agentExecOptions) ttyExecResult {
	if ctx.Err() != context.DeadlineExceeded {
		return result
	}
	result.ExitCode = 124
	result.Error = "TTY exec timed out after " + strconv.Itoa(opts.TimeoutSeconds) + "s"
	return result
}

func ttyResult(driver string, err error, stderr string) ttyExecResult {
	result := ttyExecResult{Driver: driver, Stderr: stderr, ExitCode: 0}
	if err == nil {
		return result
	}
	result.ExitCode = 1
	result.Error = err.Error()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	}
	return result
}

func shellCommand(command cruxapi.CommandSpec) string {
	parts := []string{shellQuote(command.Path)}
	for _, arg := range command.Args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func commandEnv(command cruxapi.CommandSpec) []string {
	env := os.Environ()
	envMap := make(map[string]string, len(command.Env)+1)
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			envMap[key] = value
		}
	}
	for key, value := range command.Env {
		envMap[key] = value
	}
	if dir := filepath.Dir(command.Path); dir != "." && dir != "" {
		path := envMap["PATH"]
		if path == "" {
			envMap["PATH"] = dir
		} else if !pathHasDir(path, dir) {
			envMap["PATH"] = dir + string(os.PathListSeparator) + path
		}
	}
	out := make([]string, 0, len(envMap))
	for key, value := range envMap {
		out = append(out, key+"="+value)
	}
	sort.Strings(out)
	return out
}

func pathHasDir(pathValue, dir string) bool {
	for _, part := range filepath.SplitList(pathValue) {
		if part == dir {
			return true
		}
	}
	return false
}

func ttyInput(values []string) string {
	var b strings.Builder
	for _, value := range values {
		b.WriteString(value)
		if !strings.HasSuffix(value, "\n") && !strings.HasSuffix(value, "\r") {
			b.WriteString("\r")
		}
	}
	return b.String()
}

func expectProgram(opts agentExecOptions) string {
	timeout := opts.TimeoutSeconds
	if timeout <= 0 {
		timeout = -1
	}
	var b strings.Builder
	fmt.Fprintf(&b, "set timeout %d\n", timeout)
	b.WriteString("log_user 1\n")
	b.WriteString("log_file -noappend $env(CRUX_TTY_TRANSCRIPT)\n")
	b.WriteString("set cmd [lindex $argv 0]\n")
	b.WriteString("set rest [lrange $argv 1 end]\n")
	b.WriteString("eval spawn -noecho [linsert $rest 0 $cmd]\n")
	for _, send := range opts.Send {
		fmt.Fprintf(&b, "send -- %s\n", tclQuote(send+"\r"))
	}
	if len(opts.Send) > 0 {
		b.WriteString("expect {\n")
		b.WriteString("  eof {}\n")
		b.WriteString("  timeout { exit 124 }\n")
		b.WriteString("}\n")
	} else {
		b.WriteString("interact\n")
	}
	b.WriteString("set result [wait]\n")
	b.WriteString("exit [lindex $result 3]\n")
	return b.String()
}

func tclQuote(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "$", "\\$")
	value = strings.ReplaceAll(value, "[", "\\[")
	value = strings.ReplaceAll(value, "]", "\\]")
	return "\"" + value + "\""
}

func missingExpectations(transcript string, expected []string) []string {
	missing := make([]string, 0)
	for _, value := range expected {
		if !strings.Contains(transcript, value) {
			missing = append(missing, value)
		}
	}
	return missing
}

func defaultTTYTranscriptPath(agentName string) (string, error) {
	base, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(base, ".local", "state", "crux", "tty")
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(path, cruxapi.CleanAgentName(agentName)+"-"+time.Now().UTC().Format("20060102T150405Z")+".log"), nil
}

func printExecPlan(out io.Writer, plan cruxapi.AgentExecPlan) {
	fmt.Fprintf(out, "Agent: %s\n", plan.AgentName)
	fmt.Fprintf(out, "Provider: %s\n", plan.Provider)
	fmt.Fprintf(out, "Command: %s %s\n", plan.Command.Path, strings.Join(plan.Command.Args, " "))
	if plan.Command.WorkingDir != "" {
		fmt.Fprintf(out, "WorkingDir: %s\n", plan.Command.WorkingDir)
	}
	if len(plan.Command.Env) > 0 {
		fmt.Fprintln(out, "Env:")
		keys := make([]string, 0, len(plan.Command.Env))
		for key := range plan.Command.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(out, "  %s=%s\n", key, plan.Command.Env[key])
		}
	}
}
