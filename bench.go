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


func run(ctx context.Context, log *slog.Logger, args []string) error {
	// this is the flag set for global flags
	fs := flag.NewFlagSet("bench", flag.ContinueOnError)


	// parse global flags. The subcommand must be the first non flag argument
	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("parsing arguments %w", err)

	}

	if fs.NArg() == 0 {
		return fmt.Errorf("a sub command must be provided")
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