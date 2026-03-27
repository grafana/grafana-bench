package report

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/grafana/grafana-bench/pkg/config"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/executor/trufflehog"
	"github.com/grafana/grafana-bench/pkg/executor/zizmor"
	"github.com/grafana/grafana-bench/pkg/parser/gotest"
	"github.com/grafana/grafana-bench/pkg/parser/jscoverage"
	"github.com/grafana/grafana-bench/pkg/parser/playwright"
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

			suiteRun, err := benchConfig.BuildSuiteRun(log)
			if err != nil {
				return err
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
				suiteRunSummary, err = playwright.ParseJsonOutput(log, input)
				if err != nil {
					return fmt.Errorf("parsing playwright json input %w", err)
				}
			case "go":
				suiteRunSummary, err = gotest.ParseJsonOutput(input)
				if err != nil {
					return fmt.Errorf("parsing go-json input %w", err)
				}
			case "zizmor":
				suiteRunSummary, err = zizmor.ParseSARIF(input)
				if err != nil {
					return fmt.Errorf("parsing zizmor SARIF input %w", err)
				}
			case "trufflehog":
				suiteRunSummary, err = trufflehog.ParseFindings(input)
				if err != nil {
					return fmt.Errorf("parsing trufflehog JSON input %w", err)
				}
			case "jscoverage":
				if benchConfig.JSCoverage.Codeowner == "" {
					return fmt.Errorf("missing --js-coverage-codeowner flag for jscoverage input")
				}
				if benchConfig.JSCoverage.Package == "" {
					return fmt.Errorf("missing --js-coverage-package flag for jscoverage input")
				}
				suiteRunSummary, err = jscoverage.ParseCoverageJSON(input, benchConfig.JSCoverage.Codeowner, benchConfig.JSCoverage.Package)
				if err != nil {
					return fmt.Errorf("parsing JavaScript coverage JSON input %w", err)
				}
			default:
				if benchConfig.Report.Input == "" {
					return fmt.Errorf("invalid input format - no input type specified: %q", benchConfig.Report.Input)
				} else {
					return fmt.Errorf("invalid input format %q", benchConfig.Report.Input)
				}
			}
			suiteRunSummary.SuiteName = benchConfig.TestSuite.Name
			suiteRunSummary.SuiteRevision = benchConfig.Revision

			// Inject repo label into all parser-generated metrics so every metric
			// can be filtered/grouped by repo in Grafana dashboards.
			for i := range suiteRunSummary.Metrics {
				if suiteRunSummary.Metrics[i].Labels == nil {
					suiteRunSummary.Metrics[i].Labels = make(map[string]string)
				}
				suiteRunSummary.Metrics[i].Labels["repo"] = benchConfig.TestSuite.Name
			}

			runMetrics, err := benchConfig.GetRunMetrics(log)
			if err != nil {
				return err
			}
			suiteRunSummary.Metrics = append(suiteRunSummary.Metrics, runMetrics...)

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

	config.AddBenchFlags(fs, benchConfig)
	config.AddTestTypeFlag(fs, &benchConfig.Test)
	config.AddServiceFlags(fs, &benchConfig.Service)
	config.AddSuiteNameFlag(fs, &benchConfig.TestSuite)
	config.AddSuiteRevisionFlag(fs, &benchConfig.TestSuite)
	config.AddSuiteRunFlags(fs, &benchConfig.SuiteRun)
	config.AddReportOutputFlags(fs, &benchConfig.Report)
	config.AddReportInputFlags(fs, &benchConfig.Report)
	config.AddPrometheusFlags(fs, &benchConfig.Prometheus)
	config.AddJSCoverageFlags(fs, &benchConfig.JSCoverage)

	return &cmd
}
