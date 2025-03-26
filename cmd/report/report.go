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

			suiteRunName := id.SuiteRunName(
				benchConfig.SuiteRun.Trigger,
				benchConfig.TestSuite.Name,
				benchConfig.Test.Type,
			)

			suiteRun := executor.SuiteRun{
				Name:           suiteRunName,
				Id:             runId,
				Trigger:        benchConfig.SuiteRun.Trigger,
				TestExecutor:   benchConfig.Report.Input,
				SuiteName:      benchConfig.TestSuite.Name,
				SuiteRevision:  benchConfig.TestSuite.Revision,
				BenchRevision:  benchConfig.Revision,
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

			runMetrics, err := benchConfig.GetRunMetrics()
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
	config.AddGrafanaFlags(fs, &benchConfig.Grafana)
	config.AddSuiteNameFlag(fs, &benchConfig.TestSuite)
	config.AddSuiteRevisionFlag(fs, &benchConfig.TestSuite)
	config.AddSuiteRunFlags(fs, &benchConfig.SuiteRun)
	config.AddReportOutputFlags(fs, &benchConfig.Report)
	config.AddReportInputFlags(fs, &benchConfig.Report)
	config.AddPrometheusFlags(fs, &benchConfig.Prometheus)

	return &cmd
}
