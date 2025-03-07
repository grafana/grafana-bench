package report

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/grafana/grafana-bench/pkg/config"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/executor/gotest"
	"github.com/grafana/grafana-bench/pkg/executor/playwright"
	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/revision"
	"github.com/grafana/grafana-bench/pkg/utils/id"
	"github.com/spf13/cobra"
)

const (
	long = `
report subcommand reports test suite execution results.

Presently if supports only playwright test results in json format.

It produces a human readable output or a structured log output based on the format flag.

When using the log format, in order to report the test suite execution results, the following
information is needed:
- trigger
- test type
- suite name
- suite revision (optional)
- bench revision (optional)
- grafana version

If the grafana version is not provided, the reporter will connect to the grafana instance
using the admin user and password provided and get the version.
`
	examples = `
grafana-bench report \
  --trigger local \
  --test-type smoke \
  --suite-name smoke-test \
  --report-input playwright \
  --report-output log /path/to/playwright/report.json


Configuration File
------------------

The report command supports reading configuration from a YAML file. The default file is bench.yaml.
The file can be specified using the --config flag.

The configuration file can contain any of the flags supported by the report command.

As a convention, a flag with the name "--foo-bar" in the command line will be
represented in the configuration file as:
   foo:
     bar: value

Notice that some flag names have changed to accommodate the configuration file format.

** NOTE: Deprecated flag names are not supported in the configuration file. **

Precedence: The flags specified on the command line and the environment variables will take precedence over the
values in the configuration file.

** NOTE: we strongly discourage storing sensitive information such as tokens in the configuration file. **
Consider using the environment variables or secrets manager for storing sensitive information.
Also consider using .env files for storing this information in local development environments.

# bench.yaml example
trigger: "ci"

test:
  type: "smoke"

report:
    input: "playwright"
    output: "text"

suite:
  name: "my-test-suite"
  revision: "main"
  
grafana:
  url: "http://localhost:3000"
  version: "v10.0.0"
`
)

// NewCmd returns a new bench report command
func NewCmd(log *slog.Logger) *cobra.Command {
	var benchConfig = &config.BenchConfig{}

	cmd := cobra.Command{
		Use:     "report",
		Short:   "report test suite execution results",
		Long:    long,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			if len(args) == 0 {
				return fmt.Errorf("missing report file path")
			}

			if benchConfig.TestSuite.Name == "" {
				return fmt.Errorf("missing test suite name")
			}

			grafanaSlug := ""
			if benchConfig.Grafana.Url != "" {
				// in case of error, slug it will be empty
				grafanaSlug, _ = grafana.Slug(benchConfig.Grafana.Url)
				if err != nil {
					return fmt.Errorf("failed to get grafana slug: %w", err)
				}
			}

			// get grafana version if not provided
			grafanaVersion := benchConfig.Grafana.Version
			if grafanaVersion == "" {
				if benchConfig.Grafana.Url == "" || benchConfig.Grafana.AdminUser == "" || benchConfig.Grafana.AdminPassword == "" {
					return fmt.Errorf("grafana admin user and password are needed to get grafana version")
				}

				grafanaInstance, err := grafana.NewInstance(
					benchConfig.Grafana.Url,
					benchConfig.Grafana.AdminUser,
					benchConfig.Grafana.AdminUser,
				)
				if err != nil {
					return fmt.Errorf("failed to create grafana instance: %w", err)
				}
				grafanaVersion, err = grafanaInstance.GetGrafanaBuildVersion()
				if err != nil {
					return fmt.Errorf("failed to get grafana version: %w", err)
				}
			}

			// get attributes of this test suite run using the test suite information from the config
			runId := benchConfig.SuiteRun.Id
			if runId == "" {
				runId = id.Run(benchConfig.SuiteRun.Trigger, time.Now())
			}

			suiteRunName := id.SuiteRunName(benchConfig.SuiteRun.Trigger, benchConfig.TestSuite.Name, benchConfig.Test.Type)

			suiteRun := executor.SuiteRun{
				Name:           suiteRunName,
				Id:             runId,
				Trigger:        benchConfig.SuiteRun.Trigger,
				TestExecutor:   benchConfig.Report.Input,
				SuiteName:      benchConfig.TestSuite.Name,
				SuiteRevision:  benchConfig.TestSuite.Revision,
				BenchRevision:  benchConfig.BenchRevision,
				GrafanaURL:     benchConfig.Grafana.Url,
				GrafanaSlug:    grafanaSlug,
				GrafanaVersion: grafanaVersion,

			}

			reporter, err := benchConfig.BuildReporter()
			if err != nil {
				return err
			}

			// get playwright json report
			input, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("failed to open report file: %w", err)
			}
			defer input.Close()

			var suiteRunSummary executor.SuiteRunSummary
			switch benchConfig.Report.Input {
			case "playwright":
				suiteRunSummary, err = playwright.ParseJsonOutput(input)
				if err != nil {
					return fmt.Errorf("parsing playwright json input %w", err)
				}
			case "go":
				suiteRunSummary, err = gotest.ParseJsonOutput(input)
				if err != nil {
					return fmt.Errorf("parsing go-json input %w", err)
				}
			default:
				return fmt.Errorf("invalid input format %q", benchConfig.Report.Input)
			}

			// add custom metrics adding the prefix to the name
			if suiteRunSummary.Metrics == nil {
				suiteRunSummary.Metrics = map[string]string{}
			}
			for k, v := range benchConfig.SuiteRun.Metrics {
				suiteRunSummary.Metrics[benchConfig.SuiteRun.MetricsPrefix+k] = v
			}

			err = reporter.Report(
				cmd.Context(),
				suiteRun,
				suiteRunSummary,
			)
			if err != nil {
				return fmt.Errorf("reporting test suite run %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(
		&benchConfig.Report.Output,
		"format",
		"",
		"deprecated. Use --report-output instead",
	)
	fs.StringVar(
		&benchConfig.Report.Output,
		"report-output",
		"log",
		"format of the test execution report. Allowed values 'log', 'json' and 'text'."+
			"\n log' produced a structure log. 'json' produce a json object for each log line."+
			"\n'text' produced an human readable output",
	)
	fs.StringVar(
		&benchConfig.Test.Type,
		"test-type",
		"smoke",
		"test type. Allowed values: 'smoke', 'load'",
	)
	fs.StringVar(
		&benchConfig.SuiteRun.Trigger,
		"test-trigger",
		"",
		"deprecated. Use --run-trigger instead",
	)
	fs.StringVar(
		&benchConfig.SuiteRun.Trigger,
		"trigger",
		"",
		"deprecated. Use --run-trigger instead",
	)
	fs.StringVar(
		&benchConfig.SuiteRun.Trigger,
		"run-trigger",
		"local",
		"bench execution trigger",
	)
	fs.StringVar(
		&benchConfig.BenchRevision,
		"bench-revision",
		revision.BenchRevision(),
		"bench revision. If not provided BENCH_REVISION env var is used. "+
			"\nIf not set, the current git revision is used",
	)
	fs.StringVar(
		&benchConfig.Grafana.Url,
		"grafana-url",
		"",
		"grafana url. If not provided GRAFANA_URL env var is used",
	)
	fs.StringVar(
		&benchConfig.Grafana.Version,
		"grafana-version",
		"",
		"grafana version. If not provided GRAFANA_VERSION env var is used." +
		"\nIf not set, the version is retrieved from the grafana instance, if provided.",
	)
	fs.StringVar(
		&benchConfig.Grafana.AdminUser,
		"grafana-admin-user",
		"admin",
		"grafana admin user name. Overridden by the GRAFANA_ADMIN_USER environment variable",
	)
	fs.StringVar(
		&benchConfig.Grafana.AdminPassword,
		"grafana-admin-password",
		"admin",
		"grafana admin user's password. Overridden by the GRAFANA_ADMIN_PASSWORD environment variable",
	)
	fs.StringVar(
		&benchConfig.TestSuite.Name,
		"test-suite-name",
		"",
		"deprecated. Use --suite-name instead",
	)
	fs.StringVar(
		&benchConfig.TestSuite.Name,
		"suite-name",
		"",
		"test suite name. If not specified, SUITE_NAME environment variable is used.",
	)
	fs.StringVar(
		&benchConfig.SuiteRun.Id,
		"test-suite-run",
		"",
		"deprecated. Use --run-id",
	)
	fs.StringVar(
		&benchConfig.SuiteRun.Id,
		"run-id",
		"",
		"test suite run id. If not specified, RUN_ID environment variable is used."+
			"\nIf not set, an id is generated from the execution timestamp",
	)
	fs.StringToStringVar(
		&benchConfig.SuiteRun.Metrics,
		"suite-run-metrics",
		nil,
		"deprecated use --run-metrics",
	)
	fs.StringToStringVar(
		&benchConfig.SuiteRun.Metrics,
		"run-metrics",
		nil,
		"test suite run custom metrics",
	)
	fs.StringVar(
		&benchConfig.SuiteRun.MetricsPrefix,
		"suite-run-metrics-prefix",
		"",
		"deprecated. Use --run-metrics-prefix",
	)
	fs.StringVar(
		&benchConfig.SuiteRun.MetricsPrefix,
		"run-metrics-prefix",
		"",
		"prefix to append to the suite run metric names",
	)
	fs.StringVar(
		&benchConfig.Report.Input,
		"report-input",
		"",
		"report input format. Valid values are 'playwright' and 'go'",
	)

	return &cmd
}
