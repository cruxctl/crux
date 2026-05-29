package commands

import (
	"context"
	"fmt"
)

func policyCmd() *Cmd {
	return &Cmd{
		Name:  "policy",
		Short: "Manage policies",
		Subcommands: []*Cmd{
			{
				Name: "list",
				Run: func(ctx context.Context, args []string, opts Options) error {
					c := clientFromOpts(opts)
					policies, err := c.ListPolicies(ctx)
					if err != nil {
						return err
					}
					for _, p := range policies {
						fmt.Fprintf(opts.Out, "%s (%d rules)\n", p.Metadata.ID, len(p.Rules))
					}
					return nil
				},
			},
		},
	}
}
