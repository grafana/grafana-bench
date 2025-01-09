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
		benchRevision        string
		grafanaURL           string
		grafanaAdminPassword string
		grafanaAdminUser     string
		grafanaVersion       string
		format               string
		testType             string
		trigger              string
		runId                string
		testSuiteName        string
		testSuiteRevision    string
		suiteRun             string
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
			if grafanaURL == "" {
				return fmt.Errorf("grafana url is required")
			}
			grafanaAdminUser = env.EnvOrDefault("GRAFANA_ADMIN_USER", grafanaAdminUser)
			grafanaAdminPassword = env.EnvOrDefault("GRAFANA_ADMIN_PASSWORD", grafanaAdminPassword)
			grafanaVersion = env.EnvOrDefault("GRAFANA_VERSION", grafanaVersion)

			// get grafana version if not provided
			if grafanaVersion == "" {
				if grafanaAdminUser == "" || grafanaAdminPassword == "" {
					return fmt.Errorf("grafana admin user and password are needed to get grafana version")
				}
				grafanaInstance, err := grafana.NewInstance(
					grafanaURL,
					grafanaAdminUser,
					grafanaAdminPassword,
				)
				if err != nil {
					return fmt.Errorf("failed to create grafana instance: %w", err)
				}
				grafanaVersion, err = grafanaInstance.GetGrafanaBuildVersion()
				if err != nil {
					return fmt.Errorf("failed to get grafana version: %w", err)
				}
			}

			// get playwright json report
			input, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("failed to open report file: %s", err)
			}
			defer input.Close()

			logAttrs := []any{
				"testTrigger", trigger,
				"testExecutor", playwright.ExecutorName,
				"benchRevision", benchRevision,
				"grafanaUrl", grafanaURL,
				"grafanaSlug", grafana.Slug(grafanaURL),
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
				return fmt.Errorf("parsing playwright json input %w", err)
			}

			// FIXME: we cannot reliably know the test suite base directory and path so
			// this TestSuite struct is incomplete.
			// The TestSuite struct is used by the reporters to get only the test suite name and revision
			// except for the notifications reporter, that use the base directory and path to get the codeowners
			// The Report method should be modified to accept the suite name and revision as arguments instead of the TestSuite struct
			// The Notifications reporter should be modified to accept the base directory in its constructor arguments
			// But this would limit the location of the CODEOWNERS file to the test suite base directory
			suite := executor.TestSuite{
				Name:     testSuiteName,
				Revision: testSuiteRevision,
			}

			runId = env.EnvOrDefault("RUN_ID", runId)
			if runId == "" {
				runId = id.GenRunId(time.Now(), testType)
			}

			// TODO: generate suite run id
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
		"log",
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
		&grafanaAdminUser,
		"grafana-admin-user",
		"admin",
		"grafana admin user name. Overridden by the GRAFANA_ADMIN_USER environment variable",
	)
	fs.StringVar(
		&grafanaAdminPassword,
		"grafana-admin-password",
		"admin",
		"grafana admin user's password. Overridden by the GRAFANA_ADMIN_PASSWORD environment variable",
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
