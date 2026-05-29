package commands

import (
	"context"
	"fmt"
)

func doctorCmd() *Cmd {
	return &Cmd{
		Name:  "doctor",
		Short: "Check daemon health",
		Run: func(ctx context.Context, args []string, opts Options) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: crux doctor")
			}
			cl, _, err := buildClient(opts)
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
			if opts.Format != "table" {
				return printOutput(opts, struct {
					Daemon        string `json:"daemon" yaml:"daemon"`
					Status        string `json:"status" yaml:"status"`
					ServerVersion string `json:"serverVersion" yaml:"serverVersion"`
				}{
					Daemon:        "cruxd",
					Status:        "ok",
					ServerVersion: version,
				})
			}
			fmt.Fprintf(opts.Out, "cruxd: ok (%s)\n", version)
			return nil
		},
	}
}
