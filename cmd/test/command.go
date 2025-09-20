package test

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/grafana/grafana-bench/pkg/config"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/spf13/cobra"
)

const examples = `
# run a k6 smoke test from the test suite directory
bench test --suite-path /path/to/test/folder

# run a k6 load test using a single test
bench test --test-type load --suite-path /path/to/test.js"

# checkout a test from a repo and run tests from my-branch branch
bench test \
  --suite-repo-url https://url/to/test-repo.git \
  --suite-base path/to/local/repo/directory \
  --suite-revision my-branch \
  --suite-path tests

# run k6 test with cloud output
bench test \
  --grafana-url "http://host.docker.internal:3000" \
  --suite-path /home/bench/work/grafana-plugin-tests \
  --test-runner k6
  --k6-cloud-output=true

# run k6 test with custom environment variables
bench test \
  --suite-path /home/bench/work/grafana-plugin-tests \
  --test-env VAR=value,ANOTHER_VAR=value        \
  --test-runner k6

# run playwright test
bench test  \
  --grafana-url "http://host.docker.internal:3000" \
  --suite-path grafana-plugin-tests \
  --test-runner playwright \
  --pw-prepare "yarn install" \
  --pw-execute "yarn test" \

# run go test
bench test  \
  --suite-path ./path/to/test/... \
  --test-runner go
`

const longDescription = `
test subcommand is a wrapper for running a suite of tests
against a grafana instance.

The tests to be executed are defined by the --suite-path option.

The --test-runner option defines the type of test to execute. The default is k6.

Supports two kinds of test executions defined by the --test-type option:
* smoke: execute tests and reports failures (default)
* load: execute tests and report execution stats

k6
--
Executes a test suite using k6.

For load tests, if the --k6-cloud-output flag is true, the test results will be
sent to Grafana Cloud k6. The GCK6 credentials[1] must be provided as
environment variables or as arguments (--k6-cloud-project and --k6-cloud-token)

Tests are parameterized via environment variables following grafana-api-tests
conventions[2].

[1] https://grafana.com/docs/grafana-cloud/k6/author-run/tokens-and-cli-authentication/
[2] https://github.com/grafana/grafana-api-tests/blob/main/README.md#common-environment-variables


Playwright
----------

Executes a test suite using playwright.

The --pw-prepare and --pw-execute arguments define the commands to be execute for 
preparing and executing the tests

The url to the grafana instance defined in the --grafana-url cli arguments will 
be passed to the test in the PLAYWRIGHT_BASE_URL environment variable.
See [1] for details on how to develop playwright tests compatible with the bench
test runner.

Go
--

Execute go a test suite. It wraps go test command and reports the results using the bench reporters.
It also provides the possibility of retrying failed test and report flaky tests.

The --go-test-packages argument defines the packages to be tested using the same format as the go test command.
If not specified, only the tests under the current working directory are executed.
    grafana-bench test --test-runner go --go-test-packages ./path/to/packages/...

The '--suite-path' can be used to change the working directory for the go test command.
The following command will search for tests under the 'tests' directory using the pattern defined by
the --go-test-packages:
    grafana-bench test --test-runner go \
      --suite-path tests --go-test-packages ./path/to/package/...

Additional arguments such as build tags can be passed using the --go-args flag.
    grafana-bench test --test-runner go \
       --go-test-packages ./path/to/package/... \
       --go-args "-tags=slow -race -timeout=30m"

For passing flags to configure the test, use the --go-test-args flag
    grafana-bench test --test-runner go \
       --go-test-packages ./path/to/package/... \ 
       --go-test-args "-slow 1"

The go test executor can retry failed tests. Test that pass after retrying are reported as flaky.
The number of retries is defined by the go-retries option.
    grafana-bench test --test-runner go \
       --go-test-packages ./path/to/package/... \
       --go-retries 3

[1] https://github.com/grafana/grafana-bench/blob/main/docs/writing_pw_tests.md

Slack Notifications
-------------------
If the --slack-notifications flag is set, test suite failures will be notified using slack.
Use the --slack-passing option to send notifications also for passing test suites. 

Notification will be send to the codeowners of the test. The --codeowners-mapping argument
is used to find the mapping between codeowners and slack channels.

The --slack-token argument provides the slack token. If not provided, the SLACK_TOKEN 
environment variable wil be used. This token requires channel.read, groups.read and chat.write scopes.

Configuration File
------------------

The test command supports reading configuration from a YAML file. The default file is bench.yaml.
The file can be specified using the --config flag.

The configuration file can contain any of the flags supported by the test command.

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

test:
  env:
    VAR1: "value1"
    VAR2: "value2"
  type: "smoke"
  runner: "k6"

report:
  output: "text"

suite:
  name: "my-test-suite"
  path: "/path/to/tests"
  repo: "https://github.com/org/test-repo.git"
  revision: "main"
  
grafana:
  url: "http://localhost:3000"
  admin
    user: "admin"
    password: "secret"

k6:
  cloud:
    output: true
    token: "your-token"
    project: "your-project-id"

pw:
  prepare: "npm install"
  execute: "npm test"

slack:
  notifications: true
  passing: false
  token: "xoxb-your-token"

run:
  trigger: "ci"
`

// TestFailureError represents a test suite failure (exit code 1)
type TestFailureError struct {
	message string
}

func (e TestFailureError) Error() string {
	return e.message
}

// NewCmd creates a new test command
func NewCmd(log *slog.Logger) *cobra.Command {
	var benchConfig = &config.BenchConfig{}

	cmd := cobra.Command{
		Use:     "test",
		Short:   "bench test runner",
		Long:    longDescription,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("invalid argument(s): '%s'", strings.Join(args, "', '"))
			}

			suite, err := benchConfig.BuildTestSuite(log)
			if err != nil {
				return err
			}

			suiteRun, err := benchConfig.BuildSuiteRun()
			if err != nil {
				return err
			}

			testExecutor, err := benchConfig.BuildTestExecutor(
				log,
				benchConfig.Test.Executor,
			)
			if err != nil {
				return err
			}

			reporter, err := benchConfig.BuildReporter()
			if err != nil {
				return err
			}

			// set common test execution variables
			testEnvVars := map[string]string{
				"TEST_TYPE":              benchConfig.Test.Type,
				"TEST_SUITE_REVISION":    suite.Revision,
				"GRAFANA_URL":            benchConfig.Grafana.Url,
				"GRAFANA_ADMIN_USER":     benchConfig.Grafana.AdminUser,
				"GRAFANA_ADMIN_PASSWORD": benchConfig.Grafana.AdminPassword,
			}

			// add test specific environment variables
			for k, v := range benchConfig.Test.Env {
				testEnvVars[k] = os.ExpandEnv(v)
			}

			suiteRunSummary, err := testExecutor.ExecTestSuite(
				cmd.Context(),
				*suite,
				testEnvVars,
			)
			if err != nil {
				return fmt.Errorf("executing test suite run %w", err)
			}

			runMetrics, err := benchConfig.GetRunMetrics(log)
			if err != nil {
				return err
			}
			suiteRunSummary.Metrics = append(suiteRunSummary.Metrics, runMetrics...)

			err = reporter.Report(cmd.Context(), suiteRun, suiteRunSummary)
			if err != nil {
				return fmt.Errorf("reporting test suite run %w", err)
			}

			if suiteRunSummary.Status == executor.SuiteFailed {
				return TestFailureError{"test suite failed"}
			}
			return nil
		},
	}

	fs := cmd.Flags()

	config.AddBenchFlags(fs, benchConfig)
	config.AddTestFlags(fs, &benchConfig.Test)
	config.AddTestSuiteFlags(fs, &benchConfig.TestSuite)
	config.AddSuiteRunFlags(fs, &benchConfig.SuiteRun)
	config.AddGrafanaFlags(fs, &benchConfig.Grafana)
	config.AddK6Flags(fs, &benchConfig.K6)
	config.AddPlaywrightFlags(fs, &benchConfig.Playwright)
	config.AddGoExecutorFlags(fs, &benchConfig.Go)
	config.AddSlackFlags(fs, &benchConfig.Slack)
	config.AddReportOutputFlags(fs, &benchConfig.Report)
	config.AddPrometheusFlags(fs, &benchConfig.Prometheus)

	return &cmd
}
