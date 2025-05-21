package validate

import (
	"fmt"

	"github.com/grafana/grafana-bench/pkg/config"
	"github.com/spf13/cobra"
)

// NewCmd returns a new bench version command
func NewCmd() *cobra.Command {

	cmd := cobra.Command{
		Use:     "validate-codeowners",
		Short:   "validate codeowners has associated slack mapping",
		Long:    "validates there is a channel mapping for squads in the codeowners file",
		Example: "bench validate-codeowners",
		RunE: func(cmd *cobra.Command, args []string) error {

			benchConfig := &config.BenchConfig{}
			reporter, err := benchConfig.BuildReporter()
			if err != nil {
				return err
			}

			ok, mappings, err := reporter.has_all_mappings()
			if err != nil {
				return err
			}

			if ok {
				fmt.Println("all codeowners have slack channel mappings")
			}

			strict := false
			if strict {
				return fmt.Errorf("Too many Codeowners are missing slack channel mappings")
			}

			return nil
		},
	}

	return &cmd
}
