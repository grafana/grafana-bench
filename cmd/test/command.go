package test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/cmd/test/k6"
	"github.com/grafana/grafana-bench/cmd/test/playwright"
	"github.com/grafana/grafana-bench/pkg/compile"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/notifier"
	"github.com/grafana/grafana-bench/pkg/reporter"
	"github.com/grafana/grafana-bench/pkg/revision"
	"github.com/grafana/grafana-bench/pkg/utils/env"

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
If the --slack-token argument is specified, test suite failures will be notified using slack.
Notification will be send to the codeowners of the test. The --codeowners-channel-map argument is used
to find the mapping between codeowners and slack channels.
`

// NewCmd creates a new test command
func NewCmd(log *slog.Logger) *cobra.Command {
	var (
		testEnvVars        map[string]string
		testTrigger        string
		testType           string
		testRunner         string
		reportFormat       string
		verbose            bool
		grafanaUrl         string
		grafanaUsername    string
		grafanaPassword    string
		machineSpec        string
		testSuiteName      string
                testSuiteRepo      string
		testSuiteRepoDirs  []string
		gitRepoToken       string
		testSuiteRevision  string
		testSuite          string
		revisionFile       string
		testSuiteBase      string
		grafanaTimeout     time.Duration
		benchRevision      string
		dashboardURL       string
		// k6 cloud specific flags
		k6CloudToken       string
		k6CloudProjectId   string
		k6CloudOutput      bool
		// playwright cloud specific flags
		pwPrepareCmd       string
		pwExecuteCmd       string
		// slack notifications flags
		slackNotifications bool
		slackToken         string
		codeownersMap      string
	)

	cmd := cobra.Command{
		// test-suite is a mandatory option. highlight in the help
		Use:     "test",
		Short:   "bench test runner",
		Long:    longDescription,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			trt, err := ParseTestType(testType)
			if err != nil {
				return err
			}

			if revisionFile != "" {
				testSuiteRevision, err = getTestSuiteRevision(revisionFile)
				if err != nil {
					return fmt.Errorf("getting version from file %s: %w", revisionFile, err)
				}
			}

			testSuiteRevision = env.EnvOrDefault("TEST_SUITE_REVISION", testSuiteRevision)

			if benchRevision == "" {
				benchRevision = env.EnvOrDefault("BENCH_REVISION", revision.BenchRevision())
			}

			// override grafana parameters from environment variables if they are set
			grafanaUrl = env.EnvOrDefault("GRAFANA_URL", grafanaUrl)
			grafanaUsername = env.EnvOrDefault("GRAFANA_USER", grafanaUsername)
			grafanaPassword = env.EnvOrDefault("GRAFANA_PASSWORD", grafanaPassword)

			grafanaInstance, err := getGrafanaInstance(
				log,
				grafanaUrl,
				grafanaUsername,
				grafanaPassword,
				grafanaTimeout,
			)
			if err != nil {
				return err
			}

			grafanaVersion, err := grafanaInstance.GetGrafanaBuildVersion()
			if err != nil {
				return fmt.Errorf("getting grafana version: %w", err)
			}

			if k6CloudToken == "" {
				k6CloudToken = env.EnvOrDefault("K6_CLOUD_TOKEN", "")
			}

			if k6CloudProjectId == "" {
				k6CloudProjectId = env.EnvOrDefault("K6_CLOUD_PROJECT_ID", "")
			}

			// if the name of the test suite was not given, use the last element of the test suit path as name
			if testSuiteName == "" {
				defaultTestSuiteName := strings.TrimSuffix(path.Base(testSuite), path.Ext(testSuite))
				testSuiteName = env.EnvOrDefault("TEST_SUITE_NAME", defaultTestSuiteName)
			}

			if testSuiteBase == "" {
				testSuiteBase, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("getting work directory %w", err)
				}
			}

			if testSuiteRepo != "" {
				// TODO: remove TEST_SUITE_REPO_TOKEN env variable
				gitRepoToken = env.EnvOrDefault("TEST_SUITE_REPO_TOKEN", gitRepoToken)
				gitRepoToken = env.EnvOrDefault("GIT_TOKEN", gitRepoToken)

				log.Info("checking out test suite", "repository", testSuiteRepo)

				compiler := compile.NewTestCompiler(
					log,
					testSuiteBase,
					testSuiteRepo,
					testSuiteRepoDirs,
					gitRepoToken,
					testSuiteRevision,
					[]string{},
				)

				revision, err := compiler.CompileTestSuite(context.TODO())
				if err != nil {
					return fmt.Errorf("checking out test suite: %w", err)
				}
				if testSuiteRevision == "" {
					testSuiteRevision = revision
				}
			}

			suite := executor.TestSuite{
				Name:     testSuiteName,
				BaseDir:  testSuiteBase,
				Path:     testSuite,
				Revision: testSuiteRevision,
			}

			var executor executor.TestExecutor
			if testRunner == "k6" {
				executor = k6.NewK6TestExecutor(
					log,
					verbose,
					k6CloudOutput,
					k6CloudToken,
					k6CloudProjectId,
				)
			}

			if testRunner == "playwright" {
				executor = playwright.NewPlaywrightTestExecutor(log, verbose, pwPrepareCmd, pwExecuteCmd)
			}

			runnerLog := log.With(
				"testTrigger", testTrigger,
				"benchRevision", benchRevision,
				//TODO: deprecate this attribute
				"grafanaUrl", grafanaInstance.Hostname(),
				"grafanSlug", grafanaInstance.Slug(),
				"grafanaVersion", grafanaVersion,
				"testExecutor", testRunner,
			)

			// chain of test reporters
			reporters := []reporter.SuiteRunReporter{}

			// create test reporter
			var suiteReporter reporter.SuiteRunReporter
			switch reportFormat {
			case "log": suiteReporter = reporter.NewLogReporter(runnerLog)
			case "text": suiteReporter = reporter.NewTextReporter(os.Stdout)
			default: return fmt.Errorf("invalid report format %q", revisionFile)
			}
			reporters = append(reporters, suiteReporter)

			if slackNotifications {
				slackToken = env.EnvOrDefault("SLACK_TOKEN", slackToken)
				if slackToken == "" {
					return fmt.Errorf("no slack token provided")
				}

				notifier, err := notifier.NewSlackNotifier(notifier.SlackNotifierOptions{
					Token: slackToken,
					MappingFile: codeownersMap,
					DashboardURL: dashboardURL,
				})

				if err != nil {
					return fmt.Errorf("creating slack notifier: %w", err)
				}

				reporters = append(reporters, reporter.NewNotificationReporter(notifier, reporter.NotifyAll))
			}

			runner := NewTestRunner(
				runnerLog,
				testTrigger,
				grafanaInstance,
				machineSpec,
				benchRevision,
				dashboardURL,
				executor,
				reporter.NewChainReporter(reporters...),
			)

			// ensure environment variable values are expanded
			for k,v := range testEnvVars {
				testEnvVars[k] = os.ExpandEnv(v)
			}

			return runner.Exec(cmd.Context(), trt, suite, testEnvVars)
		},
	}

	fs := cmd.Flags()
	fs.StringToStringVar(&testEnvVars, "test-env-vars", nil, "custom test environment variables")
	fs.StringVar(&testTrigger, "test-trigger", "local", "test trigger")
	fs.StringVar(&testType, "test-type", "smoke", "test type. Allowed values: 'smoke', 'load'")
	fs.StringVar(&testRunner, "test-runner", "k6", "test runner. Allowed values: 'k6', 'playwright'")
	fs.StringVar(&pwPrepareCmd, "pw-prepare-cmd", "", "command used to install dependencies for the test suite eg: \"npm install\"")
	fs.StringVar(&pwExecuteCmd, "pw-execute-cmd", "", "command used to execute the test suite eg: \"npm run test\"")
	fs.StringVar(
		&reportFormat,
		"test-report-format",
		"text",
		"format of the test execution report. Allowed values 'log' or 'text'."+
			"\n 'log' produced a structure log. 'text' produced an human readable output",
	)
	fs.BoolVar(&verbose, "verbose", false, "show test outputs")
	fs.StringVar(
		&grafanaUrl,
		"grafana-url",
		"http://localhost:3000",
		"url to grafana instance. Overridden by the GRAFANA_URL environment variable",
	)
	fs.DurationVar(
		&grafanaTimeout,
		"grafana-timeout",
		grafana.DefaultGrafanaTimeout,
		"timeout for waiting grafana to be live",
	)
	fs.StringVar(
		&grafanaUsername,
		"grafana-username",
		"admin",
		"grafana user name. Overridden by the GRAFANA_USER environment variable",
	)
	fs.StringVar(
		&grafanaPassword,
		"grafana-password",
		"admin",
		"grafana password. Overridden by the GRAFANA_PASSWORD environment variable",
	)
	fs.StringVar(&machineSpec, "machine-spec", "", "grafana instance machine spec")
	fs.StringVar(
		&revisionFile,
		"test-suite-revision-file",
		"",
		"path to a file with the test suite revision. Has precedence over test-suite-revision",
	)
	// TODO: add default value as the revision is used to generate the run id
	fs.StringVar(
		&testSuiteRepo,
		"test-suite-repo",
		"",
		"repository to get the test suite from. If not set TEST_SUITE_REPO environment variable is used." + 
			"\nIf specified, the repo will be checkout into the test-suite-base directory." +
			"\nIf test-suite-revision is specified, that revision will be checkout. Otherwise the default branch will be checkout",
		)
	fs.StringVar(
		&gitRepoToken,
		"git-repo-token",
		"",
		"authentication token for accessing git repos. If not set GIT_TOKEN environment variable is used.",
		)
	fs.StringVar(
		&gitRepoToken,
		"test-suite-repo-token",
		"",
		"authentication token for the test suite repository. If not set TEST_SUITE_REPO_TOKEN environment variable is used." +
		"\n This flag is deprecated in favor of git-repo-token.",
		)
	fs.StringSliceVar(
		&testSuiteRepoDirs,
		"test-suite-repo-dirs",
		nil,
		"Directories to checkout from test suite repo. If omitted, all folders will be checkout",
		)
	fs.StringVar(
		&testSuiteRevision,
		"test-suite-revision",
		"",
		"test suite revision. If not set TEST_SUITE_REVISION environment variable is used",
	)
	fs.StringVar(
		&benchRevision,
		"bench-revision",
		"",
		"grafana bench revision. If not set BENCH_REVISION environment variable is used.",
	)
	fs.StringVar(
		&k6CloudToken,
		"k6-cloud-token",
		"",
		"K6 cloud access token. If not set K6_CLOUD_TOKEN environment variable is used",
	)
	fs.StringVar(
		&k6CloudProjectId,
		"k6-cloud-project",
		"",
		"K6 cloud project ID. If not set K6_CLOUD_PROJECT_ID environment variable is used",
	)
	fs.BoolVar(
		&k6CloudOutput,
		"k6-cloud-output",
		false,
		"send output to GCK6. Requires setting the GCK6 project ID and access token.",
	)
	fs.StringVar(
		&dashboardURL,
		"dashboard",
		"",
		"Template for the smoke test suite execution dashboard URL."+
			"\nSupports the substitution of the following variables:"+
			"\n    SuiteRun: identifier of the suite run"+
			"\nExample: http://localhost/dashboards?run={{.SuiteRun}}",
	)
	fs.StringVar(&testSuite, "test-suite", "", "path to the tests to be executed."+
		"\nThe path must be relative to the base dir (which defaults to the current directory)."+
		"\nA single .js file or a directory can be specified."+
		"\nIf a directory is specified, all .js files in the directory and its sub-directories will be executed.")
	cmd.MarkFlagRequired("test-suite")
	fs.StringVar(
		&testSuiteBase,
		"test-suite-base",
		"",
		"base directory for searching test suites. Defaults to current directory"+
			"\nIf specified, it is prefixed to the --test-suite.",
	)
	fs.StringVar(
		&testSuiteName,
		"test-suite-name",
		"",
		"test suite name. If not specified, TEST_SUITE_NAME environment variable is used."+
			"\nDefaults to the last component of --test-suite."+
			"\nFor example --test-suite /path/to/testsuite will give a test suite name of 'testsuite'.",
	)
	fs.BoolVar(
		&slackNotifications,
		"slack-notifications",
		false,
		"send notifications to slack. Requires setting the --slack-token option or the SLACK_TOKEN environment variable.",
	)
	fs.StringVar(
		&slackToken,
		"slack-token",
		"",
		"slack token used for sending notifications. If not defined SLACK_TOKEN environment variable is used." +
		"\nThe token requires chat:write and channels:read scopes",
	)
	fs.StringVar(
		&codeownersMap,
		"codeowners-channel-map",
		"slack_teams_mapping.yaml",
		"path or url to the codeowner to slack channel mapping",
	)


	return &cmd
}

// read test suite revision from file
func getTestSuiteRevision(revisionFile string) (string, error) {
	bytes, err := os.ReadFile(revisionFile)
	if err != nil {
		return "", fmt.Errorf("getting test suite revision  from %w", err)
	}
	return strings.TrimSpace(string(bytes)), nil
}

func getGrafanaInstance(
	log *slog.Logger,
	url string,
	username string,
	password string,
	timeout time.Duration,
) (grafana.GrafanaInstance, error) {
	grafanaInstance, err := grafana.NewInstance(
		url,
		username,
		password,
		grafana.WithTimeout(timeout),
	)
	if err != nil {
		return nil, err
	}

	log.Info("Waiting for grafana server...", "address", grafanaInstance.Url())

	err = grafanaInstance.WaitForLiveGrafana(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("checking Grafana is Live... %w", err)
	}
	log.Debug("Grafana server is ready!")

	return grafanaInstance, nil

}
