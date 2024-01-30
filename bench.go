package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/grafana/grafana-bench/cmd"
	"github.com/grafana/grafana-bench/cmd/runner"

	"log/slog"
)

func main() {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	err := run(ctx, log, os.Args[1:])
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}	

// usage text
const usage = `
bench provides a CLI interface for executing diverse actions implemented as sub-commands

usage of bench:
    bench <subcommand>

subcommands:
    test   run a test suite

for help on subcommands use:
    bench <subcommand> --help|-h
`

func run(ctx context.Context, log *slog.Logger, args []string) error {
	// this is the flag set for global flags
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	// this function will be called when the help flag is passed
	fs.Usage = func() {
		fmt.Print(usage)
		fs.PrintDefaults()
	}


	// parse global flags. The subcommand must be the first non flag argument
	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("parsing arguments %w", err)

	}

	// no subcommand specified, print usage
	if fs.NArg() == 0 {
		fmt.Print(usage)
		fs.PrintDefaults()
		return nil
	}

	// flag parse stops at the first non-flag argument, Args holds the remaining args
	// the first one is the subcommand, the rest are the subcommand's arguments
	subCommand := fs.Arg(0)
	subCommandArgs := fs.Args()[1:]

	var cmd cmd.Command 
	switch subCommand {
	case "test":
		cmd, err = runner.NewTestRunnerCommand(log, subCommandArgs)
	default:
		return fmt.Errorf("unknown sub command %q", subCommand)
	}
	if err != nil {
		return fmt.Errorf("parsing arguments for subcommand %q: %w", subCommand, err)
	}

	err = cmd.Exec(ctx)
	if err != nil {
		fmt.Errorf("executing subcommand %q: %w", subCommand, err)
	}

	return nil
}