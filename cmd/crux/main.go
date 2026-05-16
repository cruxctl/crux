package main

import (
	"context"
	"os"

	"github.com/cruxctl/crux/internal/cli"
)

func main() {
	os.Exit(cli.New(os.Stdout, os.Stderr).Run(context.Background(), os.Args[1:]))
}
