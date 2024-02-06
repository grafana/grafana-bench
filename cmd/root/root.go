package root

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/grafana/grafana-bench/cmd/test"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// NewCmd returns a cobra.Command for grafana bench command
func NewCmd(log *slog.Logger) *cobra.Command {
	var (
		envFile string
	)

	rootCmd := &cobra.Command{
		Use:   "bench",
		Short: "grafana bench",
		Long:  "bench provides a CLI interface for executing diverse actions for running tests",
		// prevent the usage help to printed to stderr when an error is reported by a subcommand
		SilenceUsage: true,
		// this is needed to prevent cobra to print errors reported by subcommands in the stderror
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := godotenv.Load(envFile)

			// there was an error loading the .env file
			// do not report if it a "file not found" error for the default value
			if err != nil {
				if envFile == "" && errors.Is(err, os.ErrNotExist) {
					return nil
				}

				return fmt.Errorf("loading .env file: %w", err)
			}

			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&envFile, "env", "", "path to a file with the environment variables."+
		"\nIf none is specified and a .env files exists in the work directory, it will be used")

	rootCmd.AddCommand(test.NewCmd(log))

	return rootCmd
}
