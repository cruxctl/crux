package commands

import (
	"context"
	"fmt"

	"github.com/cruxctl/cruxd/pkg/cruxapi"
)

func versionCmd() *Cmd {
	return &Cmd{
		Name:  "version",
		Short: "Print client and server version",
		Run: func(ctx context.Context, args []string, opts Options) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: crux version")
			}
			cl, _, err := buildClient(opts)
			if err != nil {
				return err
			}
			version, err := cl.Version(ctx)
			if err != nil {
				if opts.Format != "table" {
					return printOutput(opts, struct {
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
				fmt.Fprintf(opts.Out, "crux client: %s\n", cruxapi.Version)
				fmt.Fprintf(opts.Out, "crux server: unavailable (%v)\n", err)
				return nil
			}
			if opts.Format != "table" {
				return printOutput(opts, struct {
					Client          string `json:"client" yaml:"client"`
					Server          string `json:"server" yaml:"server"`
					ServerAvailable bool   `json:"serverAvailable" yaml:"serverAvailable"`
				}{
					Client:          cruxapi.Version,
					Server:          version,
					ServerAvailable: true,
				})
			}
			fmt.Fprintf(opts.Out, "crux client: %s\n", cruxapi.Version)
			fmt.Fprintf(opts.Out, "crux server: %s\n", version)
			return nil
		},
	}
}
