package version

import (
	"fmt"

	"github.com/grafana/grafana-bench/pkg/revision"

	"github.com/spf13/cobra"
)

// NewCmd returns a new bench version command
func NewCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:     "version",
		Short:   "bench version",
		Long:    "Outputs bench version to stdout",
		Example: "bench version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(revision.BenchRevision())
			return nil
		},
	}

	return &cmd
}
