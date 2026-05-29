package commands

import (
	"context"
	"fmt"

	"github.com/cruxctl/crux/internal/agent"
	"github.com/cruxctl/crux/internal/pty"
)

func discoverCmd() *Cmd {
	return &Cmd{
		Name:  "discover",
		Short: "Discover installed coding agents",
		Run: func(ctx context.Context, args []string, opts Options) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: crux discover")
			}
			specs, err := agent.LoadBuiltinSpecs()
			if err != nil {
				return err
			}
			store, err := agent.DefaultStore()
			if err != nil {
				return err
			}
			results, err := agent.Discover(ctx, specs, pty.NewRunner(pty.NewFactory(), pty.NewNormalizer()))
			if err != nil {
				return err
			}
			states := make([]agent.AgentState, 0, len(results))
			for _, result := range results {
				if err := store.SaveAgent(result.State); err != nil {
					return err
				}
				states = append(states, result.State)
			}
			if opts.Format != "table" {
				return printOutput(opts, states)
			}
			fmt.Fprintln(opts.Out, "Discovered agents:")
			fmt.Fprintln(opts.Out)
			printAgentStateTable(opts.Out, states)
			return nil
		},
	}
}
