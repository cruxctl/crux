// Package commands defines crux's CLI command tree. Each subsystem lives in
// its own file (sessions.go, gateway.go, ...). Root() returns the canonical
// tree; the dispatcher in internal/cli walks it.
package commands

import (
	"context"
	"fmt"
	"io"

	"github.com/cruxctl/crux/internal/client"
)

type Options struct {
	Out         io.Writer
	Err         io.Writer
	Format      string
	ContextName string
	ServerURL   string
	APIKey      string
	ConfigPath  string
	Quiet       bool
}

type Cmd struct {
	Name        string
	Aliases     []string
	Short       string
	Subcommands []*Cmd
	Run         func(ctx context.Context, args []string, opts Options) error
}

// Root returns the canonical command tree. Order in this slice is also the
// help-listing order.
func Root() *Cmd {
	return &Cmd{
		Name: "crux",
		Subcommands: []*Cmd{
			versionCmd(), doctorCmd(), updateCmd(), discoverCmd(),
			daemonCmd(), contextCmd(), configCmd(),
			agentsCmd(), agentCmd(), runCmd(), psCmd(), sessionsCmd(),
			traceCmd(), eventsCmd(),
			gatewayCmd(), mcpCmd(), policyCmd(), aosCmd(), agbomCmd(),
			consoleCmd(),
			usageCmd(), machinesCmd(), auditCmd(),
		},
	}
}

func notImplCmd(name string) func(context.Context, []string, Options) error {
	return func(ctx context.Context, args []string, opts Options) error {
		fmt.Fprintf(opts.Out, "%s: not implemented yet\n", name)
		return nil
	}
}

func clientFromOpts(opts Options) *client.Client {
	url := opts.ServerURL
	if url == "" {
		url = "http://localhost:4357"
	}
	return client.New(url, opts.APIKey)
}
