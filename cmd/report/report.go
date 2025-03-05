package report

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/executor/playwright"
	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/reporter"
	"github.com/grafana/grafana-bench/pkg/utils/id"
	"github.com/spf13/cobra"
)

const (
	long = `
report subcommand reports test suite execution results.

Presently it supports only playwright test results in json format.

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
  --report-format log /path/to/playwright/report.json


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
Deprecated flag names are not supported in the configuration file.

The flags specified on the command line and the environment variables will take precedence over the
values in the configuration file.


# bench.yaml example
trigger: "ci"

test:
  type: "smoke"

report:
    format: "text"

suite:
  name: "my-test-suite"
  revision: "main"
  
grafana:
  url: "http://localhost:3000"
  admin
    user: "admin"
    password: "secret"
`
)

func readMetrics(file *os.File, metricsPrefix string) (executor.SuiteRunSummary, error) {

	data, err := io.ReadAll(file)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed reading metrics file: %s", err)
	}

	var rawMap map[string]any
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("error parsing JSON on custom metrics file: %v", err)
	}

	// Convert to map[string]string
	metrics := make(map[string]string)
	for key, value := range rawMap {

		// uppercase
		key = strings.ToUpper(key[:1]) + key[1:]
		metrics[metricsPrefix+key] = fmt.Sprintf("%v", value)
	}

	return executor.SuiteRunSummary{
		StartTime:     time.Now(),
		Status:        executor.SuitePassed,
		TestsExecuted: 1,
		TestsPassed:   1,
		TestsFailed:   0,
		TestsError:    0,
		Metrics:       metrics,
	}, nil

}

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
		metrics              map[string]string
		metricsPrefix        string
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

			if grafanaURL == "" {
				return fmt.Errorf("grafana url is required")
			}

			grafanaSlug, err := grafana.Slug(grafanaURL)
			if err != nil {
				return fmt.Errorf("failed to get grafana slug: %w", err)
			}

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
			filename := args[0]
			input, err := os.Open(filename)
			if err != nil {
				return fmt.Errorf("failed to open report file: %s", err)
			}
			defer input.Close()

			logAttrs := []any{
				"testTrigger", trigger,
				"testExecutor", playwright.ExecutorName,
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

			// THIS IS A HACK, FOR THE HACKATHON. REMOVE !!!
			// if metrics.json. assume we have key value metrics
			// and ignore playwright.
			// we took this approach because current logic expects a file to be available
			// another good approach could be to pass a report type, and a file
			var suiteRun executor.SuiteRunSummary
			if strings.Contains(filename, "metrics.json") {
				suiteRun, err = readMetrics(input, metricsPrefix)
				if err != nil {
					return err
				}
			} else {
				suiteRun, err = playwright.ParseJsonOutput(input)
				if err != nil {
					return fmt.Errorf("parsing playwright json input %w", err)
				}
			}

			// add custom metrics adding the prefix to the name
			if suiteRun.Metrics == nil {
				suiteRun.Metrics = map[string]string{}
			}
			for k, v := range metrics {
				suiteRun.Metrics[metricsPrefix+k] = v
			}

			if runId == "" {
				runId = id.Run(trigger, time.Now())
			}

			err = suiteReporter.Report(
				cmd.Context(),
				testSuiteName,
				testSuiteRevision,
				runId,
				id.SuiteRunName(trigger, testSuiteName, testType),
				suiteRun,
			)
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
		"",
		"deprecated. Use --report-format instead",
	)
	fs.StringVar(
		&format,
		"report-format",
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
		"",
		"deprecated. Use --trigger instead",
	)
	fs.StringVar(
		&trigger,
		"trigger",
		"local",
		"bench execution trigger",
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
		"deprecated. Use --suite-name instead",
	)
	fs.StringVar(
		&testSuiteName,
		"suite-name",
		"",
		"test suite name. If not specified, SUITE_NAME environment variable is used.",
	)
	fs.StringVar(
		&suiteRun,
		"test-suite-run",
		"",
		"deprecated. Use --suite-run-id instead",
	)
	fs.StringVar(
		&suiteRun,
		"suite-run-id",
		"",
		"test suite run id. If not specified, SUITE_RUN_ID environment variable is used."+
			"\nIf not set, an id is generated from the execution timestamp",
	)
	fs.StringToStringVar(
		&metrics,
		"suite-run-metrics",
		nil,
		"test suite run custom metrics",
	)
	fs.StringVar(
		&metricsPrefix,
		"suite-run-metrics-prefix",
		"",
		"prefix to append to the suite run metric names",
	)

	return &cmd
}
