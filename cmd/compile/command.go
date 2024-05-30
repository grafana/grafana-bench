package compile

import (
	"log/slog"
	"strings"

	"github.com/grafana/grafana-bench/pkg/compile"
	"github.com/spf13/cobra"
)

const examples = `
bench compile --target-dir work/tests  \
    --test-suite-repo git@github.com:grafana/grafana-api-tests \
    --test-suite-revision main
`

// NewCmd returns a new test compile command
func NewCmd(log *slog.Logger) *cobra.Command {
	log = log.With("svc", "test-compiler")
	var (
		targetDir         string
		testSuiteRepo     string
		repoToken         string
		testSuiteRevision string
		prepareCmd        string
	)

	cmd := cobra.Command{
		Use:     "compile",
		Short:   "bench test compiler",
		Long:    "bench compile subcommand retrieves and builds a test suite from a given source location",
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdArgs := []string{}
			if prepareCmd != "" {
				cmdArgs = strings.Split(prepareCmd, " ")
			}

			compiler := compile.NewTestCompiler(
				log,
				targetDir,
				testSuiteRepo,
				repoToken,
				testSuiteRevision,
				cmdArgs,
			)

			return compiler.CompileTestSuite(cmd.Context())
		},
	}

	fs := cmd.Flags()
	// FIXME: find a better name
	fs.StringVar(&targetDir, "target-dir", "", "directory for checking the test into.")
	fs.StringVar(&testSuiteRepo, "test-suite-repo", "", "repository to grab test suite from")
	fs.StringVar(&repoToken, "test-suite-repo-token", "", "access token for the repository")
	fs.StringVar(&testSuiteRevision, "test-suite-revision", "", "test suite revision to compile."+
		"\nCan make reference to a branch (local or remote), a tag or a specific commit hash"+
		"\nIf not provided and the repo is already checked out in the base dir, the current branch is compiled."+
		"\nOtherwise the main branch from the remote repository is compiled")
	fs.StringVar(&prepareCmd, "prepare-command", "", "command to execute to prepare the test e.g 'npm install")

	return &cmd
}
