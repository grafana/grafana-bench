package test

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/runner"
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
`

const longDescription = `
test subcommand is a wrapper for running a suite of k6 or playwright tests
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
Deprecated flag names are not supported in the configuration file.

The flags specified on the command line and the environment variables will take precedence over the
values in the configuration file.

NOTE: we strongly discourage storing sensitive information such as tokens in the configuration file.
Consider using the environment variables or secrets manager for storing sensitive information.
Also consider using .env files for storing this information in local development environments.

# bench.yaml example
trigger: "ci"

test:
  env:
    VAR1: "value1"
    VAR2: "value2"
  type: "smoke"
  runner: "k6"

report:
  format: "text"

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
`

// NewCmd creates a new test command
func NewCmd(log *slog.Logger) *cobra.Command {
	var (	
		config      = &BenchConfig{}
		suiteConfig = &TestSuiteConfig{}
	)

	cmd := cobra.Command{
		Use:     "test",
		Short:   "bench test runner",
		Long:    longDescription,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("invalid argument(s): '%s'", strings.Join(args, "', '"))
			}

			suite, err := suiteConfig.BuildTestSuite(log, config.BaseDir)
			if err != nil {
				return err
			}

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
	fs.StringToStringVar(
		&config.EnvVars,
		"test-env-vars",
		nil,
		"deprecated. Use test-env",
	)
	fs.StringToStringVar(
		&config.EnvVars,
		"test-env",
		nil,
		"environment variables passed to the test execution.",
	)
	fs.StringVar(
		&config.Trigger,
		"test-trigger",
		"local",
		"deprecated. Use trigger",
	)
	fs.StringVar(
		&config.Trigger,
		"trigger",
		"local",
		"trigger of bench execution. For example, 'ci' or 'local'.",
	)
	fs.StringVar(
		&config.Type,
		"test-type",
		"smoke",
		"test type. Allowed values: 'smoke', 'load'",
	)
	fs.StringVar(
		&suiteConfig.TestExecutor,
		"test-runner",
		"k6",
		"test runner. Allowed values: 'k6', 'playwright'",
	)
	fs.StringVar(
		&config.PW.PrepareCmd,
		"pw-prepare-cmd",
		"",
		"deprecated. Use pw-prepare",
	)
	fs.StringVar(
		&config.PW.PrepareCmd,
		"pw-prepare",
		"",
		"commands used to install dependencies for the test suite eg: \"npm install\"."+
			"\nMultiple commands can be specified by separating with ';'.",
	)
	fs.StringVar(
		&config.PW.ExecuteCmd,
		"pw-execute-cmd",
		"",
		"deprecated. Use pw-execute",
	)
	fs.StringVar(
		&config.PW.ExecuteCmd,
		"pw-execute",
		"",
		"command used to execute the test suite eg: \"npm run test\"",
	)
	fs.StringVar(
		&config.ReportFormat,
		"test-report-format",
		"",
		"deprecated. Use report-format",
	)
	fs.StringVar(
		&config.ReportFormat,
		"report-format",
		"text",
		"format of the test execution report. Allowed values 'log' or 'text'."+
			"\n 'log' produced a structure log. 'text' produced an human readable output",
	)
	fs.BoolVar(
		&config.Verbose,
		"verbose",
		false,
		"show test outputs",
	)
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
		&config.Grafana.AdminUser,
		"grafana-admin-user",
		"admin",
		"grafana admin user name. Overridden by the GRAFANA_ADMIN_USER environment variable",
	)
	fs.StringVar(
		&config.Grafana.AdminPassword,
		"grafana-admin-password",
		"admin",
		"grafana admin user's password. Overridden by the GRAFANA_ADMIN_PASSWORD environment variable",
	)
	// TODO: add default value as the revision is used to generate the run id
	fs.StringVar(
		&suiteConfig.Repo,
		"test-suite-repo",
		"",
		"deprecated. Use suite-repo-url",
	)
	fs.StringVar(
		&suiteConfig.Repo,
		"suite-repo-url",
		"",
		"url to the repository to get the test suite from. If not set SUITE_REPO_URL environment variable is used."+
			"\nIf specified, the repo will be checkout into the --suite-base directory."+
			"\nIf --suite-revision is specified, that revision will be checkout."+
			"\nOtherwise the default branch will be checkout",
	)
	fs.StringVar(
		&suiteConfig.RepoToken,
		"test-suite-repo-token",
		"",
		"deprecated. Use suite-repo-token",
		)
	fs.StringVar(
		&suiteConfig.RepoToken,
		"suite-repo-token",
		"",
		"authentication token for the test suite repository. "+
			"\nIf not set SUITE_REPO_TOKEN environment variable is used.",
	)
	fs.StringSliceVar(
		&suiteConfig.RepoDirs,
		"test-suite-repo-dirs",
		nil,
		"deprecated. Use suite-repo-dirs",
	)
	fs.StringSliceVar(
		&suiteConfig.RepoDirs,
		"suite-repo-dirs",
		nil,
		"Directories to checkout from test suite repo. If omitted, all folders will be checkout",
	)
	fs.StringVar(
		&suiteConfig.Revision,
		"test-suite-revision",
		"",
		"deprecated. Use suite-revision",
	)
	fs.StringVar(
		&suiteConfig.Revision,
		"suite-revision",
		"",
		"test suite revision. If not set SUITE_REVISION environment variable is used",
	)
	fs.StringVar(
		&config.BenchRevision,
		"bench-revision",
		config.BenchRevision,
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
		"k6-cloud-project-id",
		"",
		"deprecated. Use k6-cloud-project",
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
	fs.StringVar(
		&suiteConfig.Path,
		"test-suite",
		"",
		"deprecated. Use suite-path")
	fs.StringVar(
		&suiteConfig.Path,
		"suite-path",
		"",
		"path to the tests to be executed."+
			"\nThe path must be relative to the base dir (which defaults to the current directory)."+
			"\nA single .js file or a directory can be specified."+
			"\nIf a directory is specified, all files in the directory and its sub-directories will be executed.")
	fs.StringVar(
		&config.BaseDir,
		"test-suite-base",
		"",
		"deprecated. Use suite-base",
	)
	fs.StringVar(
		&config.BaseDir,
		"suite-base",
		"",
		"base directory for searching test suites. Defaults to current directory"+
			"\nIf specified, it is prefixed to the --suite-path.",
	)
	fs.StringVar(
		&suiteConfig.Name,
		"test-suite-name",
		"",
		"deprecated. Use suite-name",
	)
	fs.StringVar(
		&suiteConfig.Name,
		"suite-name",
		"",
		"test suite name. If not specified, SUITE_NAME environment variable is used."+
			"\nDefaults to the last component of -suite-path."+
			"\nFor example --suite--path path/to/testsuite will give a test suite name of 'testsuite'.",
	)
	fs.BoolVar(
		&config.NotifyPassing,
		"notify-passing",
		false,
		"deprecated. Use slack-notify-passing",
	)
	fs.BoolVar(
		&config.NotifyPassing,
		"slack-passing",
		false,
		"send notifications for passing test suites. By default only not passing test suites are notified",
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
		"codeowners-mapping",
		"codeowners-mapping.yaml",
		"deprecated. Use slack-codeowners-mapping")
	fs.StringVar(
		&config.Slack.CodeownersMap,
		"slack-codeowners-mapping",
		"codeowners-mapping.yaml",
		"path or url to the codeowner to slack channel id mapping."+
			"\nRelative to test suite base dir.",
	)

	return &cmd
}
