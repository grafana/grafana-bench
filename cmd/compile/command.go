package compile

import (
	"log/slog"
	"strings"

	"github.com/grafana/grafana-bench/pkg/compile"
	"github.com/grafana/grafana-bench/pkg/utils/env"
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
		targetDir          string
                testSuiteRepo      string
		testSuiteRepoDirs  []string
		testSuiteRepoToken string
		testSuiteRevision  string
		prepareCmd         string
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

			testSuiteRepoToken = env.EnvOrDefault("TEST_SUITE_REPO_TOKEN", testSuiteRepoToken )

			compiler := compile.NewTestCompiler(
				log,
				targetDir,
				testSuiteRepo,
				testSuiteRepoDirs,
				testSuiteRepoToken,
				testSuiteRevision,
				cmdArgs,
			)

			revision, err := compiler.CompileTestSuite(cmd.Context())
			if err != nil {
				return err
			}

			log.Info("checkout", "revision", revision)
			return nil
		},
	}

	fs := cmd.Flags()
	// FIXME: find a better name
	fs.StringVar(&targetDir, "target-dir", "", "directory for checking the test into.")
	// TODO: add default value as the revision is used to generate the run id
	fs.StringVar(
		&testSuiteRepo,
		"test-suite-repo",
		"",
		"repository to get the test suite from. If not set TEST_SUITE_REPO environment variable is used." + 
			"\nIf specified, the repo will be checkout into the test-suite-base directory." +
			"\nIf test-suite-revision is specified, that revision will be checkout. Otherwise the default branch will be checkout",
		)
	fs.StringVar(
		&testSuiteRepoToken,
		"test-suite-repo-token",
		"",
		"authentication token for the test suite repository. If not set TEST_SUITE_REPO_TOKEN environment variable is used.",
		)
	fs.StringSliceVar(
		&testSuiteRepoDirs,
		"test-suite-repo-dirs",
		nil,
		"Directories to checkout from test suite repo. If omitted, all directories will be checkout",
		)
	fs.StringVar(
		&testSuiteRevision,
		"test-suite-revision",
		"",
		"test suite revision. If not set TEST_SUITE_REVISION environment variable is used",
	)
	fs.StringVar(&prepareCmd, "prepare-command", "", "command to execute to prepare the test e.g 'npm install")

	return &cmd
}
