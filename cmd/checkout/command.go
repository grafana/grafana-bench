package checkout

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/grafana/grafana-bench/pkg/compile"
	"github.com/grafana/grafana-bench/pkg/config"
	"github.com/spf13/cobra"
)

const examples = `
# checkout a test from a repo and run tests from my-branch branch
bench checkout \
  --suite-repo-url https://url/to/test-repo.git \
  --suite-base path/to/local/repo/directory \
  --suite-revision my-branch 
`

const longDescription = `
checkout subcommand gets the test source from a git repository.
`

// NewCmd creates a new test command
func NewCmd(log *slog.Logger) *cobra.Command {
	var benchConfig = &config.BenchConfig{}

	cmd := cobra.Command{
		Use:     "checkout",
		Short:   "bench test source checkout",
		Long:    longDescription,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("invalid argument(s): '%s'", strings.Join(args, "', '"))
			}

			compiler := compile.NewTestCompiler(
				log,
				benchConfig.Git.Driver,
				benchConfig.TestSuite.BaseDir,
				benchConfig.TestSuite.Repo,
				benchConfig.TestSuite.RepoDirs,
				benchConfig.TestSuite.RepoToken,
				benchConfig.TestSuite.Revision,
				[]string{},
			)

			_, err := compiler.CompileTestSuite(context.TODO())
			if err != nil {
				return fmt.Errorf("checking out test suite: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	config.AddGitFlags(fs, &benchConfig.Git)
	config.AddSuiteRepoFlags(fs, &benchConfig.TestSuite)
	config.AddSuiteRevisionFlag(fs, &benchConfig.TestSuite)

	// we only need the suite base
	fs.StringVar(
		&benchConfig.TestSuite.BaseDir,
		"suite-base",
		".",
		"base directory for searching test suites. Defaults to current directory"+
			"\nIf specified, it is prefixed to the --suite-path.",
	)

	return &cmd
}
