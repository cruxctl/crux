package commands

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/cruxctl/crux/internal/cli/console"
)

func consoleCmd() *Cmd {
	return &Cmd{
		Name:    "console",
		Short:   "Manage the web console",
		Aliases: []string{"ui"},
		Subcommands: []*Cmd{
			{Name: "start", Short: "Start the console server", Run: consoleStart},
			{Name: "stop", Short: "Stop the console server", Run: consoleStop},
			{Name: "status", Short: "Check console status", Run: consoleStatus},
			{Name: "open", Short: "Open console in browser", Run: consoleOpen},
		},
	}
}

func consoleStart(ctx context.Context, args []string, opts Options) error {
	var port int
	var dev bool
	fs := flag.NewFlagSet("console start", flag.ContinueOnError)
	fs.SetOutput(opts.Err)
	fs.IntVar(&port, "port", 4358, "console port")
	fs.BoolVar(&dev, "dev", false, "run in development mode")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return console.Start(ctx, console.Options{
		Out:    opts.Out,
		Err:    opts.Err,
		Port:   port,
		Dev:    dev,
		APIURL: opts.ServerURL,
	})
}

func consoleStop(ctx context.Context, args []string, opts Options) error {
	return console.Stop(ctx, console.Options{Out: opts.Out, Err: opts.Err})
}

func consoleStatus(ctx context.Context, args []string, opts Options) error {
	var port int
	fs := flag.NewFlagSet("console status", flag.ContinueOnError)
	fs.SetOutput(opts.Err)
	fs.IntVar(&port, "port", 4358, "console port")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return console.Status(ctx, console.Options{Out: opts.Out, Port: port})
}

func consoleOpen(ctx context.Context, args []string, opts Options) error {
	var port int
	fs := flag.NewFlagSet("console open", flag.ContinueOnError)
	fs.SetOutput(opts.Err)
	fs.IntVar(&port, "port", 4358, "console port")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return console.Open(ctx, console.Options{Out: opts.Out, Port: port})
}

func ConsoleUsage(out io.Writer) {
	fmt.Fprintln(out, `Usage:
  crux console start [--port PORT] [--dev]
  crux console stop
  crux console status [--port PORT]
  crux console open [--port PORT]

Manage the Crux Console web UI.`)
}
