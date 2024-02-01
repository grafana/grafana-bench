package main

import (
	"context"
	"os"

	"github.com/grafana/grafana-bench/cmd/runner"
	"github.com/spf13/cobra"

	"log/slog"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	root := newRootCmd()
	root.AddCommand(runner.NewTestRunnerCommand(log))

	err := root.ExecuteContext(context.Background())
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}	

func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bench",
		Short: "grafana bench",
		Long: "bench provides a CLI interface for executing diverse actions for running tests",
		// prevent the usage help to printed to stderr when an error is reported by a subcommand
		SilenceUsage:  true,
		// this is needed to prevent cobra to print errors reported by subcommands in the stderror
		SilenceErrors: true,
	}
}