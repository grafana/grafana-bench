package report

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/executor/playwright"
	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/reporter"
	"github.com/grafana/grafana-bench/pkg/revision"
	"github.com/grafana/grafana-bench/pkg/utils/env"
	"github.com/grafana/grafana-bench/pkg/utils/id"
	"github.com/spf13/cobra"
)

const (
	long = `
report subcommand reports test suite execution results.

Presently if supports only playwright test results in json format.

It produces a human readable output or a structured log output based on the format flag.
`
	examples = `
grafana-bench report --format log /path/to/playwright/report.json
`
)

// NewCmd returns a new bench report command
func NewCmd(log *slog.Logger) *cobra.Command {
	var (
		benchRevision     string
		grafanaURL        string
		grafanaVersion    string
		format            string
		testType	  string
		trigger           string
		runId             string
		testSuiteName     string
		testSuiteRevision string
		suiteRun          string
	)
	cmd := cobra.Command{
		Use:     "report",
		Short:   "report test suite execution results",
		Long:    long,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing report file path")
			}

			if benchRevision == "" {
				benchRevision = env.EnvOrDefault("BENCH_REVISION", revision.BenchRevision())
			}

			grafanaURL = env.EnvOrDefault("GRAFANA_URL", grafanaURL)
			grafanaVersion = env.EnvOrDefault("GRAFANA_VERSION", grafanaVersion)
			grafanaSlug := grafana.Slug(grafanaURL)

			input, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("failed to open report file: %s", err)
			}
			defer input.Close()

			logAttrs := []any{
				"testTrigger", trigger,
				"testExecutor", "playwrigh",
				"benchRevision", benchRevision,
				"grafanaUrl", grafanaURL,
				"grafanaSlug", grafanaSlug,
				"grafanaVersion", grafanaVersion,
			}

			var suiteReporter reporter.SuiteRunReporter
			switch format {
			case "log":
				suiteReporter = reporter.NewLogReporter(logAttrs)
			case "text":
				suiteReporter = reporter.NewTextReporter(os.Stdout)
			default:
				return fmt.Errorf("invalid report format %q", format)
			}

			suiteRun, err := playwright.ParseJsonOutput(input)
			if err != nil {
				return fmt.Errorf("parsing input %w", err)
			}

			suite := executor.TestSuite{
				Name: testSuiteName,
				Revision: testSuiteRevision,
			}

			runId = env.EnvOrDefault("RUN_ID", runId)
			if runId == "" {
				runId = id.GenRunId(time.Now(), testType)
			}

			err = suiteReporter.Report(cmd.Context(), runId, "suiteRunId", suite, suiteRun)
			if err != nil {
				return fmt.Errorf("reporting test suite run %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()

	fs.StringVar(
		&format,
		"format",
		"text",
		"format of the test execution report. Allowed values 'log' or 'text'."+
			"\n 'log' produced a structure log. 'text' produced an human readable output",
	)
	fs.StringVar(
		&testType,
		"test-type",
		"smoke",
		"test type. Allowed values: 'smoke', 'load'",
	)
	fs.StringVar(
		&trigger,
		"test-trigger",
		"local",
		"test trigger",
	)
	fs.StringVar(
		&benchRevision,
		"bench-revision",
		"",
		"bench revision. If not provided BENCH_REVISION env var is used. "+
			"\nIf not set, the current git revision is used",
	)
	fs.StringVar(
		&grafanaURL,
		"grafana-url",
		"",
		"grafana url. If not provided GRAFANA_URL env var is used",
	)
	fs.StringVar(
		&grafanaVersion,
		"grafana-version",
		"",
		"grafana version. If not provided GRAFANA_VERSION env var is used",
	)
	fs.StringVar(
		&testSuiteName,
		"test-suite-name",
		"",
		"test suite name. If not specified, TEST_SUITE_NAME environment variable is used.",
	)
	fs.StringVar(
		&suiteRun,
		"test-suite-run",
		"",
		"test suite run id. If not specified, TEST_SUITE_NAME environment variable is used."+
		"\nIf not set, an id is generated from the execution timestamp",
	)

	return &cmd
}
