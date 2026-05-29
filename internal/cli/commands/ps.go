package commands

import (
	"context"
	"fmt"

	"github.com/cruxctl/crux/internal/agent"
	"github.com/cruxctl/crux/internal/pty"
)

func psCmd() *Cmd {
	return &Cmd{
		Name:    "ps",
		Short:   "List discovered coding agents",
		Aliases: []string{},
		Run: func(ctx context.Context, args []string, opts Options) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: crux ps")
			}
			store, err := agent.DefaultStore()
			if err != nil {
				return err
			}
			states, err := store.ListAgents()
			if err != nil {
				return err
			}
			if len(states) == 0 {
				specs, loadErr := agent.LoadBuiltinSpecs()
				if loadErr != nil {
					return loadErr
				}
				results, discoverErr := agent.Discover(ctx, specs, pty.NewRunner(pty.NewFactory(), pty.NewNormalizer()))
				if discoverErr != nil {
					return discoverErr
				}
				for _, result := range results {
					if err := store.SaveAgent(result.State); err != nil {
						return err
					}
					states = append(states, result.State)
				}
			}
			if opts.Format != "table" {
				return printOutput(opts, states)
			}
			printAgentStateTable(opts.Out, states)
			return nil
		},
	}
}
