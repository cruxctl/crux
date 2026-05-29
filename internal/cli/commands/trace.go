package commands

import (
	"context"
	"fmt"

	"github.com/cruxctl/crux/internal/client"
)

func traceCmd() *Cmd {
	return &Cmd{
		Name:  "trace",
		Short: "Show events for an execution",
		Run: func(ctx context.Context, args []string, opts Options) error {
			if len(args) != 1 {
				return fmt.Errorf("usage: crux trace <execution-id|last>")
			}
			cl, _, err := buildClient(opts)
			if err != nil {
				return err
			}
			id := args[0]
			if id == "last" {
				executions, err := cl.ListExecutions(ctx)
				if err != nil {
					return err
				}
				if len(executions) == 0 {
					return fmt.Errorf("no executions found")
				}
				id = executions[0].ID
			}
			events, err := cl.Events(ctx, id)
			if err != nil {
				return err
			}
			if opts.Format != "table" {
				return printOutput(opts, events)
			}
			printEventTable(opts.Out, events, false)
			return nil
		},
	}
}

func eventsCmd() *Cmd {
	return &Cmd{
		Name:  "events",
		Short: "Show all daemon events",
		Run: func(ctx context.Context, args []string, opts Options) error {
			if len(args) > 1 || (len(args) == 1 && args[0] != "ls") {
				return fmt.Errorf("usage: crux events [ls]")
			}
			cl, _, err := buildClient(opts)
			if err != nil {
				return err
			}
			events, err := cl.Events(ctx, "")
			if err != nil {
				return err
			}
			if opts.Format != "table" {
				return printOutput(opts, events)
			}
			printEventTable(opts.Out, events, true)
			return nil
		},
	}
}

var _ = client.New // silence unused import if no other client usage
