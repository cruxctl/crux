package commands

import (
	"context"
	"fmt"
)

func gatewayCmd() *Cmd {
	return &Cmd{
		Name:  "gateway",
		Short: "Manage Crux Gateway",
		Subcommands: []*Cmd{
			{
				Name: "status",
				Run: func(ctx context.Context, args []string, opts Options) error {
					c := clientFromOpts(opts)
					status, err := c.GetGatewayStatus(ctx)
					if err != nil {
						return err
					}
					fmt.Fprintf(opts.Out, "enabled=%v ready=%v\n", status.Enabled, status.Ready)
					return nil
				},
			},
		},
	}
}
