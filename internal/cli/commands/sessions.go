package commands

import (
	"context"
	"fmt"
)

func sessionsCmd() *Cmd {
	return &Cmd{
		Name:  "sessions",
		Short: "Manage agent sessions",
		Subcommands: []*Cmd{
			{
				Name: "list",
				Run: func(ctx context.Context, args []string, opts Options) error {
					c := clientFromOpts(opts)
					sessions, err := c.ListSessions(ctx)
					if err != nil {
						return err
					}
					for _, s := range sessions {
						fmt.Fprintf(opts.Out, "%s %s %s\n", s.ID, s.AgentID, s.Status)
					}
					return nil
				},
			},
			{
				Name: "stop",
				Run: func(ctx context.Context, args []string, opts Options) error {
					if len(args) != 1 {
						return fmt.Errorf("usage: crux sessions stop <id>")
					}
					c := clientFromOpts(opts)
					if err := c.StopSession(ctx, args[0]); err != nil {
						return err
					}
					fmt.Fprintln(opts.Out, "session stopped")
					return nil
				},
			},
		},
	}
}
