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
	"github.com/cruxctl/crux/internal/cli/commands"
	"github.com/cruxctl/crux/internal/config"
	"github.com/cruxctl/crux/internal/logging"
	"github.com/cruxctl/crux/pkg/cruxapi"
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
	verbose    bool
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
			if !c.agentCommandUsage(cmd, nil) {
				fmt.Fprintf(c.err, "unknown command %q\n", cmd)
				return 1
			}
		}
		return 0
	}
	if nestedHelpArg(cmd, rest) {
		if !c.commandUsage(cmd, []string{rest[0]}) {
			if !c.agentCommandUsage(cmd, []string{rest[0]}) {
				fmt.Fprintf(c.err, "unknown command %q\n", cmd)
				return 1
			}
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
	case "discover":
		runErr = c.discoverAgents(ctx, opts, rest)
	case "ps":
		runErr = c.psAgents(ctx, opts, rest)
	case "trace":
		runErr = c.trace(ctx, opts, rest)
	case "events":
		runErr = c.events(ctx, opts, rest)
	default:
		// Try the canonical command tree first, then fall back to agent-scoped dispatch.
		if treeCmd, subRest := lookupCommand(commands.Root(), cmd, rest); treeCmd != nil && treeCmd.Run != nil {
			runErr = treeCmd.Run(ctx, subRest, c.toCmdOpts(opts))
		} else {
			runErr = c.agentScoped(ctx, opts, cmd, rest)
		}
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

func lookupCommand(root *commands.Cmd, name string, args []string) (*commands.Cmd, []string) {
	for _, sc := range root.Subcommands {
		if sc.Name == name {
			if len(args) == 0 {
				return sc, args
			}
			for _, ssc := range sc.Subcommands {
				if ssc.Name == args[0] {
					return ssc, args[1:]
				}
			}
			return sc, args
		}
	}
	return nil, args
}

func (c *CLI) toCmdOpts(opts rootOptions) commands.Options {
	return commands.Options{
		Out:         c.out,
		Err:         c.err,
		Format:      opts.output,
		ContextName: opts.context,
		ServerURL:   opts.serverURL,
		APIKey:      opts.apiKey,
		ConfigPath:  opts.configPath,
		Verbose:     opts.verbose,
	}
}

func (c *CLI) usage() {
	fmt.Fprintln(c.out, "Crux Control")
	fmt.Fprintln(c.out)
	fmt.Fprintln(c.out, "Usage:")
	fmt.Fprintln(c.out, "  crux [global flags] <command> [args]")
	fmt.Fprintln(c.out, "  crux <command> [args] [global flags]")
	fmt.Fprintln(c.out, "  crux <agent-name> <describe|usage|exec|conversations>")
	fmt.Fprintln(c.out)
	fmt.Fprintln(c.out, "Global flags:")
	fmt.Fprintln(c.out, "  --config PATH      CLI config file (default ~/.config/crux/config.yaml)")
	fmt.Fprintln(c.out, "  --context NAME     CLI context name")
	fmt.Fprintln(c.out, "  --server URL       cruxd server URL override")
	fmt.Fprintln(c.out, "  --api-key KEY      API key override")
	fmt.Fprintln(c.out, "  -o, --output FMT   table, json, or yaml")
	fmt.Fprintln(c.out, "  --log-level LEVEL  debug, info, warn, or error")
	fmt.Fprintln(c.out, "  --log-file PATH    CLI log file override; \"none\" disables file logging")
	fmt.Fprintln(c.out, "  -v, --verbose      Print stats after agent commands")
	fmt.Fprintln(c.out)
	fmt.Fprintln(c.out, "Commands:")
	for _, cmd := range commands.Root().Subcommands {
		name := cmd.Name
		if len(cmd.Aliases) > 0 {
			name += " (" + strings.Join(cmd.Aliases, ", ") + ")"
		}
		fmt.Fprintf(c.out, "  %-18s %s\n", name, cmd.Short)
	}
	fmt.Fprintln(c.out)
	fmt.Fprintln(c.out, "Agent commands:")
	fmt.Fprintln(c.out, "  crux claude-code describe")
	fmt.Fprintln(c.out, "  crux gemini-cli usage")
	fmt.Fprintln(c.out, "  crux aider exec --repo .")
	fmt.Fprintln(c.out, "  crux opencode conversations ls")
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

Discover installed coding agents from built-in YAML specs and store local state under ~/.crux/state.`)
	case "ps":
		fmt.Fprintln(c.out, `Usage:
  crux ps

List discovered coding agents.`)
	case "trace":
		fmt.Fprintln(c.out, `Usage:
  crux trace <execution-id|last>

Show events for an execution.`)
	case "events":
		fmt.Fprintln(c.out, `Usage:
  crux events [ls]

Show all daemon events.`)
	case "daemon":
		fmt.Fprintln(c.out, `Usage:
  crux daemon <start|stop|restart|status|logs>

Manage the cruxd daemon.`)
	case "agents":
		fmt.Fprintln(c.out, `Usage:
  crux agents <list|get|register|unregister>

Manage registered agents.`)
	case "agent":
		fmt.Fprintln(c.out, `Usage:
  crux agent <describe|usage|exec|conversations>

Operate a single agent.`)
	case "run":
		fmt.Fprintln(c.out, `Usage:
  crux run <agent> [--repo DIR] [-- PROVIDER_ARGS...]

Run a task through an agent.`)
	case "sessions":
		fmt.Fprintln(c.out, `Usage:
  crux sessions [ls|get <id>]

List or inspect PTY sessions.`)
	case "gateway":
		fmt.Fprintln(c.out, `Usage:
  crux gateway <status|config>

Manage gateway settings.`)
	case "mcp":
		fmt.Fprintln(c.out, `Usage:
  crux mcp <list|add|remove>

Manage MCP servers.`)
	case "policy":
		fmt.Fprintln(c.out, `Usage:
  crux policy <list|get|apply>

Manage policies.`)
	case "aos":
		fmt.Fprintln(c.out, `Usage:
  crux aos <status|events>

AOS operations.`)
	case "agbom":
		fmt.Fprintln(c.out, `Usage:
  crux agbom [view]

View AgBOM.`)
	case "console":
		commands.ConsoleUsage(c.out)
	case "usage":
		fmt.Fprintln(c.out, `Usage:
  crux usage [summary]

View usage and costs.`)
	case "machines":
		fmt.Fprintln(c.out, `Usage:
  crux machines [ls]

Manage machines.`)
	case "audit":
		fmt.Fprintln(c.out, `Usage:
  crux audit [ls]

View audit logs.`)
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

func helpArg(args []string) bool {
	return len(args) > 0 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help")
}

func nestedHelpArg(command string, args []string) bool {
	if len(args) < 2 || (args[1] != "--help" && args[1] != "-h" && args[1] != "help") {
		return false
	}
	switch command {
	case "config", "context", "daemon", "agents", "agent", "gateway", "mcp", "policy", "aos", "sessions":
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
	fs.BoolVar(&opts.verbose, "verbose", false, "verbose output with stats")
	fs.BoolVar(&opts.verbose, "v", false, "verbose output with stats")
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
	}, os.Stderr)
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

func currentWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
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
