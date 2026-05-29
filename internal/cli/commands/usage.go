package commands

import (
	"context"
	"fmt"
)

func usageCmd() *Cmd {
	return &Cmd{
		Name:  "usage",
		Short: "View usage and limits",
		Subcommands: []*Cmd{
			{
				Name: "show",
				Run: func(ctx context.Context, args []string, opts Options) error {
					c := clientFromOpts(opts)
					usage, err := c.GetUsage(ctx, "", "", "")
					if err != nil {
						return err
					}
					for _, u := range usage {
						fmt.Fprintf(opts.Out, "%s: $%.4f (%d sessions)\n", u.AgentID, u.USD, u.Sessions)
					}
					return nil
				},
			},
			{
				Name: "limits",
				Run: func(ctx context.Context, args []string, opts Options) error {
					c := clientFromOpts(opts)
					limits, err := c.GetUsageLimits(ctx)
					if err != nil {
						return err
					}
					fmt.Fprintf(opts.Out, "warn=$%.2f block=$%.2f\n", limits.DailyUSDWarn, limits.DailyUSDBlock)
					return nil
				},
			},
		},
	}
}
