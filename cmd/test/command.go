package test

import (
	"log/slog"
	"os"

	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/runner"

	"github.com/spf13/cobra"
)

const examples = `
# run a k6 smoke test from the test suite directory
bench test --test-suite /path/to/test/folder

# run a k6 load test using a single test
bench test --test-type load --test-suite /path/to/test.js"

# checkout a test from a repo and run tests from my-branch branch
bench test \
  --test-suite-repo https://url/to/test-repo.git \
  --test-suite-base path/to/local/repo/directory \
  --test-suite-revision my-branch \
  --test-suite tests

# run k6 test with cloud output
bench test \
  --grafana-url "http://host.docker.internal:3000" \
  --test-suite /home/bench/work/grafana-plugin-tests \
  --test-runner k6
  --k6-cloud-output=true

# run k6 test with custom environment variables
bench test \
  --test-suite /home/bench/work/grafana-plugin-tests \
  --test-env-vars VAR=value,ANOTHER_VAR=value        \
  --test-runner k6

# run playwright test
bench test  \
  --grafana-url "http://host.docker.internal:3000" \
  --test-suite grafana-plugin-tests \
  --test-runner playwright \
  --pw-prepare-cmd "yarn install" \
  --pw-execute-cmd "yarn test" \
`

const longDescription = `
test subcommand is a wrapper for running a suite of k6 or playwright tests
against a grafana instance.

The tests to be executed are defined by the --test-suite option.

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


[1] https://github.com/grafana/grafana-bench/blob/main/docs/writing_pw_tests.md

Slack Notifications
-------------------
If the --slack-notifications flag is set, test suite failures will be notified using slack.
Notification will be send to the codeowners of the test. The --codeowners-channel-map argument is used
to find the mapping between codeowners and slack channels.

The --slack-token argument provides the slack token. If not provided, the SLACK_TOKEN 
environment variable wil be used. This token requires channel.read, groups.read and chat.write scopes.
`

// NewCmd creates a new test command
func NewCmd(log *slog.Logger) *cobra.Command {
	var (
		config      = &BenchConfig{}
		suiteConfig = &TestSuiteConfig{}
	)

	cmd := cobra.Command{
		// test-suite is a mandatory option. highlight in the help
		Use:     "test",
		Short:   "bench test runner",
		Long:    longDescription,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			suiteConfig.MergeEnv()

			suite, err := suiteConfig.BuildTestSuite(log)
			if err != nil {
				return err
			}

			config.MergeEnv()
			testRunner, err := config.BuildTestRunner(log, suiteConfig.TestExecutor)
			if err != nil {
				return err
			}

			// ensure environment variable values are expanded
			testEnvVars := map[string]string{}
			for k, v := range config.EnvVars {
				testEnvVars[k] = os.ExpandEnv(v)
			}

			trt, err := runner.ParseTestType(config.Type)
			if err != nil {
				return err
			}

			return testRunner.Exec(cmd.Context(), trt, *suite, testEnvVars)
		},
	}

	fs := cmd.Flags()
	fs.StringToStringVar(&config.EnvVars, "test-env-vars", nil, "custom test environment variables")
	fs.StringVar(&config.Trigger, "test-trigger", "local", "test trigger")
	fs.StringVar(&config.Type, "test-type", "smoke", "test type. Allowed values: 'smoke', 'load'")
	fs.StringVar(&suiteConfig.TestExecutor, "test-runner", "k6", "test runner. Allowed values: 'k6', 'playwright'")
	fs.StringVar(&config.PW.PrepareCmd, "pw-prepare-cmd", "", "command used to install dependencies for the test suite eg: \"npm install\"")
	fs.StringVar(&config.PW.ExecuteCmd, "pw-execute-cmd", "", "command used to execute the test suite eg: \"npm run test\"")
	fs.StringVar(
		&config.ReportFormat,
		"test-report-format",
		"text",
		"format of the test execution report. Allowed values 'log' or 'text'."+
			"\n 'log' produced a structure log. 'text' produced an human readable output",
	)
	fs.BoolVar(&config.Verbose, "verbose", false, "show test outputs")
	fs.StringVar(
		&config.Grafana.Url,
		"grafana-url",
		"http://localhost:3000",
		"url to grafana instance. Overridden by the GRAFANA_URL environment variable",
	)
	fs.DurationVar(
		&config.Grafana.Timeout,
		"grafana-timeout",
		grafana.DefaultGrafanaTimeout,
		"timeout for waiting grafana to be live",
	)
	fs.StringVar(
		&config.Grafana.UserName,
		"grafana-username",
		"admin",
		"grafana user name. Overridden by the GRAFANA_USER environment variable",
	)
	fs.StringVar(
		&config.Grafana.Password,
		"grafana-password",
		"admin",
		"grafana password. Overridden by the GRAFANA_PASSWORD environment variable",
	)
	// TODO: add default value as the revision is used to generate the run id
	fs.StringVar(
		&suiteConfig.Repo,
		"test-suite-repo",
		"",
		"repository to get the test suite from. If not set TEST_SUITE_REPO environment variable is used."+
			"\nIf specified, the repo will be checkout into the test-suite-base directory."+
			"\nIf test-suite-revision is specified, that revision will be checkout. Otherwise the default branch will be checkout",
	)
	fs.StringVar(
		&suiteConfig.RepoToken,
		"test-suite-repo-token",
		"",
		"authentication token for the test suite repository. If not set TEST_SUITE_REPO_TOKEN environment variable is used.",
	)
	fs.StringSliceVar(
		&suiteConfig.RepoDirs,
		"test-suite-repo-dirs",
		nil,
		"Directories to checkout from test suite repo. If omitted, all folders will be checkout",
	)
	fs.StringVar(
		&suiteConfig.Revision,
		"test-suite-revision",
		"",
		"test suite revision. If not set TEST_SUITE_REVISION environment variable is used",
	)
	fs.StringVar(
		&config.BenchRevision,
		"bench-revision",
		"",
		"grafana bench revision. If not set BENCH_REVISION environment variable is used.",
	)
	fs.StringVar(
		&config.K6.CloudToken,
		"k6-cloud-token",
		"",
		"K6 cloud access token. If not set K6_CLOUD_TOKEN environment variable is used",
	)
	fs.StringVar(
		&config.K6.CloudProjectId,
		"k6-cloud-project",
		"",
		"K6 cloud project ID. If not set K6_CLOUD_PROJECT_ID environment variable is used",
	)
	fs.BoolVar(
		&config.K6.CloudOutput,
		"k6-cloud-output",
		false,
		"send output to GCK6. Requires setting the GCK6 project ID and access token.",
	)
	fs.StringVar(
		&config.DashboardURL,
		"dashboard",
		"",
		"Template for the smoke test suite execution dashboard URL."+
			"\nSupports the substitution of the following variables:"+
			"\n    SuiteRun: identifier of the suite run"+
			"\nExample: http://localhost/dashboards?run={{.SuiteRun}}",
	)
	fs.StringVar(&suiteConfig.Path, "test-suite", "", "path to the tests to be executed."+
		"\nThe path must be relative to the base dir (which defaults to the current directory)."+
		"\nA single .js file or a directory can be specified."+
		"\nIf a directory is specified, all .js files in the directory and its sub-directories will be executed.")
	//cmd.MarkFlagRequired("test-suite")
	fs.StringVar(
		&suiteConfig.BaseDir,
		"test-suite-base",
		"",
		"base directory for searching test suites. Defaults to current directory"+
			"\nIf specified, it is prefixed to the --test-suite.",
	)
	fs.StringVar(
		&suiteConfig.Name,
		"test-suite-name",
		"",
		"test suite name. If not specified, TEST_SUITE_NAME environment variable is used."+
			"\nDefaults to the last component of --test-suite."+
			"\nFor example --test-suite /path/to/testsuite will give a test suite name of 'testsuite'.",
	)
	fs.BoolVar(
		&config.SlackNotifications,
		"slack-notifications",
		false,
		"send notifications to slack. Requires setting the --slack-token option or the SLACK_TOKEN environment variable.",
	)
	fs.StringVar(
		&config.Slack.Token,
		"slack-token",
		"",
		"slack token used for sending notifications. If not defined SLACK_TOKEN environment variable is used."+
			"\nThe token requires chat:write and channels:read scopes",
	)
	fs.StringVar(
		&config.Slack.CodeownersMap,
		"codeowners-channel-map",
		"slack_teams_mapping.yaml",
		"path or url to the codeowner to slack channel mapping",
	)

	return &cmd
}
