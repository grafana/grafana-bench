package compile

import (
	"log/slog"

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
		testSuiteDir      string
		testSuiteRepo     string
		testSuiteRevision string
		forceCheckout     bool
	)

	cmd := cobra.Command{
		Use:     "compile",
		Short:   "bench test compiler",
		Long:    "bench compile subcommand retrieves and builds a test suite from a given source location",
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			compiler := NewTestCompiler(
				log,
				testSuiteDir,
				testSuiteRepo,
				testSuiteRevision,
				forceCheckout,
			)

			return compiler.CompileTestSuite(cmd.Context())
		},
	}

	fs := cmd.Flags()
	// FIXME: find a better name
	fs.StringVar(
		&testSuiteDir,
		"test-suite-dir", 
		"",
		"directory where the test suite repo is checkout."+
		"\nIf the director exists and the test-suite-repo is specified, the repo will be checkout" +
		"\nonly if the force-checkout option is set to true",
	)
	fs.StringVar(
		&testSuiteRepo,
		"test-suite-repo", 
		"",
		"repository to check out test suite from.",
	)
	fs.StringVar(
		&testSuiteRevision,
		"test-suite-revision",
		"",
		"test suite revision to compile."+
		"\nCan make reference to a branch (local or remote), a tag or a specific commit hash"+
		"\nIf not provided and the repo is already checked out in the base dir, the current branch is compiled."+
		"\nOtherwise the main branch from the remote repository is compiled.",
	)
	fs.BoolVar(
		&forceCheckout,
		"force-checkout",
		false,
		"if the the test-suite-repo is specified and the test-suite-dir exists, indicates if the repo must be checkout",
	)

	return &cmd
}
