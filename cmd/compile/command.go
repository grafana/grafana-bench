package compile

import (
	"log/slog"

	"github.com/spf13/cobra"
)

// NewCmd returns a new test compile command
func NewCmd(log *slog.Logger) *cobra.Command {
	log = log.With("svc", "test-compiler")
	var (
		baseDir           string
		testSuiteRepo     string
		testSuiteRevision string
	)

	cmd := cobra.Command{
		Use:     "compile",
		Short:   "bench test compiler",
		Long:    "bench compile subcommand retrieves and builds a test suite from a given source location",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			compiler := NewTestCompiler(
				log,
				baseDir,
				testSuiteRepo,
				testSuiteRevision,
			)

			return compiler.CompileTestSuite(cmd.Context())
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&baseDir,"base-dir", "", "base directory for compiling test into. Defaults to current work directory")
	fs.StringVar(&testSuiteRepo, "test-suite-repo", "", "repository to grab test suite from")
	fs.StringVar(&testSuiteRevision, "test-suite-revision", "", "test suite revision to compile." + 
		"\nIf not provided, the latest from 'main' branch is compiled.")

	return &cmd
}