package commands

import (
	"context"
	"fmt"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

func machinesCmd() *Cmd {
	return &Cmd{
		Name:  "machines",
		Short: "Manage enrolled machines",
		Subcommands: []*Cmd{
			{
				Name: "list",
				Run: func(ctx context.Context, args []string, opts Options) error {
					c := clientFromOpts(opts)
					machines, err := c.ListMachines(ctx)
					if err != nil {
						return err
					}
					for _, m := range machines {
						fmt.Fprintf(opts.Out, "%s %s\n", m.ID, m.Status)
					}
					return nil
				},
			},
			{
				Name: "pair",
				Run: func(ctx context.Context, args []string, opts Options) error {
					if len(args) != 1 {
						return fmt.Errorf("usage: crux machines pair <token>")
					}
					c := clientFromOpts(opts)
					resp, err := c.PairMachine(ctx, cruxapi.EnrollmentRequest{Token: args[0]})
					if err != nil {
						return err
					}
					fmt.Fprintf(opts.Out, "paired: %s (%s)\n", resp.MachineID, resp.Status)
					return nil
				},
			},
		},
	}
}
