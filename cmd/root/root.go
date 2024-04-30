package root

import (
	"errors"
	"fmt"
	"os"

	"github.com/grafana/grafana-bench/cmd/compile"
	"github.com/grafana/grafana-bench/cmd/test"
	"github.com/grafana/grafana-bench/cmd/version"
	"github.com/grafana/grafana-bench/pkg/utils/env"
	"github.com/grafana/grafana-bench/pkg/utils/logger"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// NewCmd returns a cobra.Command for grafana bench command
func NewCmd(log *logger.Logger) *cobra.Command {
	var (
		envFile string
		logLevel string
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
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("loading .env file: %w", err)
			}

			logLevel = env.EnvOrDefault("BENCH_LOG_LEVEL", logLevel)
			err = log.ParseLevel(logLevel)
			if err != nil {
				return err
			}

			return nil
		},
	}

	pf := rootCmd.PersistentFlags()
	pf.StringVar(&envFile, "env", "", "path to a file with the environment variables."+
		"\nIf none is specified and a .env files exists in the work directory, it will be used")
	pf.StringVar(&logLevel, "log-level", "ERROR", "set the log level ('ERROR', 'WARN', 'INFO', 'DEBUG')." +
		"\n overridden by the BENCH_LOG_LEVEL environment variable")

	rootCmd.AddCommand(test.NewCmd(log.Log()))
	rootCmd.AddCommand(compile.NewCmd(log.Log()))
	rootCmd.AddCommand(version.NewCmd())

	return rootCmd
}
