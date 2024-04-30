package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/grafana/grafana-bench/cmd/root"
	"github.com/grafana/grafana-bench/pkg/utils/logger"
)

func main() {
	log := logger.NewLogger()

	docCmd := newCmd(log)

	err := docCmd.Execute()
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

const examples = `
  gendoc -o docs/  # generates markdown documentation in the docs directory
`
// creates a cobra command for doc generation
func newCmd(log *logger.Logger) *cobra.Command {
	var dir string

	cmd :=  &cobra.Command{
		Use:   "gendoc",
		Short: "grafana bench documentation generator",
		Long:  "gendoc generates documentation for the grafana bench command",
		Example: examples,
		// print usage help to stderr when an error is reported by a subcommand
		SilenceUsage: false,
		// this is needed to prevent cobra to print errors reported by subcommands in the stderror
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			benchCmd := root.NewCmd(log)
			return  doc.GenMarkdownTree(benchCmd, dir)
		},
	}
	fs := cmd.Flags()
	fs.StringVarP(&dir, "out-dir", "o", "", "output directory")
	cmd.MarkFlagRequired("out-dir")

	return cmd
}
