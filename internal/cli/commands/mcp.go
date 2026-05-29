package commands

import (
	"context"
	"fmt"
)

func mcpCmd() *Cmd {
	return &Cmd{
		Name:  "mcp",
		Short: "Manage MCP servers",
		Subcommands: []*Cmd{
			{
				Name: "list",
				Run: func(ctx context.Context, args []string, opts Options) error {
					c := clientFromOpts(opts)
					servers, err := c.ListMCPServers(ctx)
					if err != nil {
						return err
					}
					for _, s := range servers {
						fmt.Fprintf(opts.Out, "%s %s\n", s.Name, s.Status)
					}
					return nil
				},
			},
		},
	}
}
