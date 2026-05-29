package commands

import (
	"context"
	"fmt"
)

func agentsCmd() *Cmd {
	return &Cmd{
		Name:    "agents",
		Aliases: []string{"agent"},
		Short:   "List or manage agents",
		Subcommands: []*Cmd{
			{
				Name: "list",
				Run: func(ctx context.Context, args []string, opts Options) error {
					c := clientFromOpts(opts)
					agents, err := c.ListAgents(ctx)
					if err != nil {
						return err
					}
					if opts.Format != "table" {
						return printOutput(opts, agents)
					}
					fmt.Fprintln(opts.Out, "Agents:")
					for _, a := range agents {
						fmt.Fprintf(opts.Out, "  %s (%s)\n", a.Name, a.Status)
					}
					return nil
				},
			},
			{
				Name: "refresh",
				Run: func(ctx context.Context, args []string, opts Options) error {
					c := clientFromOpts(opts)
					if err := c.RefreshAgents(ctx); err != nil {
						return err
					}
					fmt.Fprintln(opts.Out, "agents refreshed")
					return nil
				},
			},
		},
	}
}
