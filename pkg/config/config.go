// Package config defines the bench configuration
package config

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/pkg/compile"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/executor/gotest"
	"github.com/grafana/grafana-bench/pkg/executor/k6"
	"github.com/grafana/grafana-bench/pkg/executor/playwright"
	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/metrics"
	"github.com/grafana/grafana-bench/pkg/notifier"
	"github.com/grafana/grafana-bench/pkg/reporter"
	"github.com/grafana/grafana-bench/pkg/revision"
	"github.com/grafana/grafana-bench/pkg/utils/id"
	"github.com/spf13/pflag"
)

type BenchConfig struct {
	// FIXME: moved there because is needed by the slack notifications
	// for the codeowners mapping.
	Revision   string
	TestSuite  TestSuiteConfig
	Test       TestConfig
	Report     ReportConfig
	SuiteRun   SuiteRunConfig
	Grafana    GrafanaConfig
	Go         GoTestConfig
	K6         K6Config
	Playwright PWConfig
	Slack      SlackNotifierConfig
	Prometheus Prometheus
}

func AddBenchFlags(fs *pflag.FlagSet, config *BenchConfig) {
	fs.StringVar(
		&config.Revision,
		"bench-revision",
		revision.BenchRevision(),
		"grafana bench revision. If not set BENCH_REVISION environment variable is used."+
			"\nIf not set, the current git revision is used (default (devel) ",
	)
}

type GrafanaConfig struct {
	Version       string
	Url           string
	AdminUser     string
	AdminPassword string
	Timeout       time.Duration
}

func AddGrafanaFlags(fs *pflag.FlagSet, config *GrafanaConfig) {
	fs.StringVar(
		&config.Url,
		"grafana-url",
		"http://localhost:3000",
		"url to grafana instance. Overridden by the GRAFANA_URL environment variable (default http://localhost:3000)",
	)
	fs.DurationVar(
		&config.Timeout,
		"grafana-timeout",
		grafana.DefaultGrafanaTimeout,
		"timeout for waiting grafana to be live",
	)
	fs.StringVar(
		&config.AdminUser,
		"grafana-admin-user",
		"admin",
		"grafana admin user name. Overridden by the GRAFANA_ADMIN_USER environment variable",
	)
	fs.StringVar(
		&config.AdminPassword,
		"grafana-admin-password",
		"admin",
		"grafana admin user's password. Overridden by the GRAFANA_ADMIN_PASSWORD environment variable",
	)
	fs.StringVar(
		&config.Version,
		"grafana-version",
		"",
		"grafana version. If not provided GRAFANA_VERSION env var is used."+
			"\nIf not set, the version is retrieved from the grafana instance.",
	)
}

type K6Config struct {
	CloudToken     string
	CloudProjectId string
	CloudOutput    bool
}

func AddK6Flags(fs *pflag.FlagSet, config *K6Config) {
	fs.StringVar(
		&config.CloudToken,
		"k6-cloud-token",
		"",
		"K6 cloud access token. If not set K6_CLOUD_TOKEN environment variable is used",
	)
	fs.StringVar(
		&config.CloudProjectId,
		"k6-cloud-project-id",
		"",
		"deprecated. Use k6-cloud-project",
	)
	fs.StringVar(
		&config.CloudProjectId,
		"k6-cloud-project",
		"",
		"K6 cloud project ID. If not set K6_CLOUD_PROJECT_ID environment variable is used",
	)
	fs.BoolVar(
		&config.CloudOutput,
		"k6-cloud-output",
		false,
		"send output to GCK6. Requires setting the GCK6 project ID and access token.",
	)
}

type PWConfig struct {
	PrepareCmd string
	ExecuteCmd string
}

func AddPlaywrightFlags(fs *pflag.FlagSet, config *PWConfig) {
	fs.StringVar(
		&config.PrepareCmd,
		"pw-prepare-cmd",
		"",
		"deprecated. Use pw-prepare",
	)
	fs.StringVar(
		&config.PrepareCmd,
		"pw-prepare",
		"",
		"commands used to install dependencies for the test suite eg: \"npm install\"."+
			"\nMultiple commands can be specified by separating with ';'.",
	)
	fs.StringVar(
		&config.ExecuteCmd,
		"pw-execute-cmd",
		"",
		"deprecated. Use pw-execute",
	)
	fs.StringVar(
		&config.ExecuteCmd,
		"pw-execute",
		"",
		"command used to execute the test suite eg: \"npm run test\"",
	)
}

type GoTestConfig struct {
	GoArgs   []string
	TestArgs []string
	Packages []string
	Retries  int
}

func AddGoExecutorFlags(fs *pflag.FlagSet, config *GoTestConfig) {
	fs.StringArrayVar(
		&config.GoArgs,
		"go-args",
		nil,
		"arguments to be passed to go test command (e.g '-tag slow -race')",
	)
	fs.IntVar(
		&config.Retries,
		"go-retries",
		0,
		"number of retries for failed tests. Retried tests that pass are reported as flaky",
	)
	fs.StringArrayVar(
		&config.TestArgs,
		"go-test-args",
		nil,
		"arguments to be passed to the test using the arg flag (e.g '-args -slow 1')",
	)
	fs.StringArrayVar(
		&config.Packages,
		"go-test-packages",
		nil,
		"patterns for selecting packages for testing. Can be repeated to specify multiple packages."+
			"\nIf no pattern is specified only tests under the current working directory are executed.",
	)
}

type SuiteRunConfig struct {
	Trigger       string
	Id            string
	DashboardURL  string
	Metrics       []string
	MetricsPrefix string
	MetricsFile   string
}

func AddSuiteRunFlags(fs *pflag.FlagSet, config *SuiteRunConfig) {
	fs.StringVar(
		&config.DashboardURL,
		"dashboard",
		"",
		"deprecated. Use run-dashboard",
	)
	fs.StringVar(
		&config.DashboardURL,
		"run-dashboard",
		"",
		"Template for the suite run dashboard URL."+
			"\nSupports the substitution of the following variables:"+
			"\n    Id: identifier of the suite run"+
			"\nExample: http://localhost/dashboards?run={{.Id}}",
	)
	fs.StringVar(
		&config.Trigger,
		"test-trigger",
		"local",
		"deprecated. Use run-trigger",
	)
	fs.StringVar(
		&config.Trigger,
		"trigger",
		"local",
		"deprecated. Use run-trigger",
	)
	fs.StringVar(
		&config.Trigger,
		"run-trigger",
		"local",
		"trigger of bench execution. For example, 'ci' or 'local'.",
	)
	fs.StringSliceVar(
		&config.Metrics,
		"suite-run-metrics",
		nil,
		"deprecated use --run-metrics",
	)
	fs.StringArrayVar(
		&config.Metrics,
		"run-metric",
		nil,
		"test suite run custom metrics. Format: name{label=label-value,..}=value. The value must be a valid float number.",
	)
	fs.StringVar(
		&config.MetricsPrefix,
		"suite-run-metrics-prefix",
		"",
		"deprecated. Use --run-metrics-prefix",
	)
	fs.StringVar(
		&config.MetricsPrefix,
		"run-metrics-prefix",
		"",
		"prefix to append to the suite run metric names",
	)
	fs.StringVar(
		&config.MetricsFile,
		"run-metrics-file",
		"",
		"path to a file containing a list of metrics to be added to the suite run."+
			"\nThe file must follow prometheus exposition format. [1]"+
			"\nEach non commented line should follow the pattern metric{label1=value1,label2=value2,...} value."+
			"\nThe timestamp, if present, is omitted and all metrics are reported using the suite run's execution time."+
			"\n[1] https://github.com/Showmax/prometheus-docs/blob/master/content/docs/instrumenting/exposition_formats.md",
	)
}

type ReportConfig struct {
	Output string
	Input  string
}

func AddReportOutputFlags(fs *pflag.FlagSet, report *ReportConfig) {
	fs.StringVar(
		&report.Output,
		"format",
		"",
		"deprecated. Use report-output",
	)
	fs.StringVar(
		&report.Output,
		"test-report-format",
		"",
		"deprecated. Use report-output",
	)
	fs.StringVar(
		&report.Output,
		"report-format",
		"text",
		"deprecated. Use report-output",
	)
	fs.StringVar(
		&report.Output,
		"report-output",
		"text",
		"format of the test execution report. Allowed values 'log' or 'text'."+
			"\n 'log' produced a structure log. 'text' produced an human readable output",
	)
}

func AddReportInputFlags(fs *pflag.FlagSet, config *ReportConfig) {
	fs.StringVar(
		&config.Input,
		"report-input",
		"",
		"report input format. Valid values are 'playwright' and 'go'",
	)
}

type TestConfig struct {
	Verbose  bool
	Type     string
	Executor string
	Env      map[string]string
}

func AddTestFlags(fs *pflag.FlagSet, test *TestConfig) {
	AddTestEnvFlags(fs, test)
	AddTestTypeFlag(fs, test)
	AddTestRunnerFlag(fs, test)
}

func AddTestEnvFlags(fs *pflag.FlagSet, test *TestConfig) {
	fs.StringToStringVar(
		&test.Env,
		"test-env-vars",
		nil,
		"deprecated. Use test-env",
	)
	fs.StringToStringVar(
		&test.Env,
		"test-env",
		nil,
		"environment variables passed to the test execution.",
	)
}

func AddTestTypeFlag(fs *pflag.FlagSet, test *TestConfig) {
	fs.StringVar(
		&test.Type,
		"test-type",
		"smoke",
		"test type. Allowed values: 'smoke', 'load'",
	)
}

func AddTestRunnerFlag(fs *pflag.FlagSet, test *TestConfig) {
	fs.StringVar(
		&test.Executor,
		"test-runner",
		"k6",
		"test runner. Allowed values: 'k6', 'playwright', 'go'",
	)
}

func AddTestVeboseFlag(fs *pflag.FlagSet, test *TestConfig) {
	fs.BoolVar(
		&test.Verbose,
		"test-verbose",
		false,
		"show test output",
	)
	fs.BoolVar(
		&test.Verbose,
		"verbose",
		false,
		"deprecated. Use verbose",
	)
}

type TestSuiteConfig struct {
	GitToken  string
	Name      string
	Repo      string
	RepoToken string
	RepoDirs  []string
	BaseDir   string
	Path      string
	Revision  string
}

func AddTestSuiteFlags(fs *pflag.FlagSet, config *TestSuiteConfig) {
	AddSuiteNameFlag(fs, config)
	AddSuitePathFlags(fs, config)
	AddSuiteRepoFlags(fs, config)
	AddSuiteRevisionFlag(fs, config)
}

func AddSuiteNameFlag(fs *pflag.FlagSet, config *TestSuiteConfig) {
	fs.StringVar(
		&config.Name,
		"test-suite-name",
		"",
		"deprecated. Use suite-name",
	)
	fs.StringVar(
		&config.Name,
		"suite-name",
		"",
		"test suite name. If not specified, SUITE_NAME environment variable is used."+
			"\nDefaults to the last component of -suite-path."+
			"\nFor example --suite--path path/to/testsuite will give a test suite name of 'testsuite'.",
	)
}

func AddSuitePathFlags(fs *pflag.FlagSet, config *TestSuiteConfig) {
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
		&config.Path,
		"test-suite",
		"",
		"deprecated. Use suite-path")
	fs.StringVar(
		&config.Path,
		"suite-path",
		"",
		"path to the tests to be executed."+
			"\nThe path must be relative to the base dir (which defaults to the current directory)."+
			"\nA single .js file or a directory can be specified."+
			"\nIf a directory is specified, all files in the directory and its sub-directories will be executed.",
	)
}

func AddSuiteRepoFlags(fs *pflag.FlagSet, config *TestSuiteConfig) {
	fs.StringVar(
		&config.Repo,
		"test-suite-repo",
		"",
		"deprecated. Use suite-repo-url",
	)
	fs.StringVar(
		&config.Repo,
		"suite-repo-url",
		"",
		"url to the repository to get the test suite from. If not set SUITE_REPO_URL environment variable is used."+
			"\nIf specified, the repo will be checkout into the --suite-base directory."+
			"\nIf --suite-revision is specified, that revision will be checkout."+
			"\nOtherwise the default branch will be checkout",
	)
	fs.StringVar(
		&config.RepoToken,
		"test-suite-repo-token",
		"",
		"deprecated. Use suite-repo-token",
	)
	fs.StringVar(
		&config.RepoToken,
		"suite-repo-token",
		"",
		"authentication token for the test suite repository. "+
			"\nIf not set SUITE_REPO_TOKEN environment variable is used.",
	)
	fs.StringSliceVar(
		&config.RepoDirs,
		"test-suite-repo-dirs",
		nil,
		"deprecated. Use suite-repo-dirs",
	)
	fs.StringSliceVar(
		&config.RepoDirs,
		"suite-repo-dirs",
		nil,
		"Directories to checkout from test suite repo. If omitted, all folders will be checkout",
	)
}

func AddSuiteRevisionFlag(fs *pflag.FlagSet, config *TestSuiteConfig) {
	fs.StringVar(
		&config.Revision,
		"test-suite-revision",
		"",
		"deprecated. Use suite-revision",
	)
	fs.StringVar(
		&config.Revision,
		"suite-revision",
		"",
		"test suite revision. If not set SUITE_REVISION environment variable is used",
	)
}

type SlackNotifierConfig struct {
	CodeownersMap string
	Token         string
	Notifications bool
	NotifyPassing bool
}

func AddSlackFlags(fs *pflag.FlagSet, config *SlackNotifierConfig) {
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
		&config.Notifications,
		"slack-notifications",
		false,
		"send notifications to slack. Requires setting the --slack-token option or the SLACK_TOKEN environment variable.",
	)
	fs.StringVar(
		&config.Token,
		"slack-token",
		"",
		"slack token used for sending notifications. If not defined SLACK_TOKEN environment variable is used."+
			"\nThe token requires chat:write and channels:read scopes",
	)
	fs.StringVar(
		&config.CodeownersMap,
		"codeowners-mapping",
		"codeowners-mapping.yaml",
		"deprecated. Use slack-codeowners-mapping")
	fs.StringVar(
		&config.CodeownersMap,
		"slack-codeowners-mapping",
		"codeowners-mapping.yaml",
		"path or url to the codeowner to slack channel id mapping."+
			"\nRelative to test suite base dir.",
	)
}

type Prometheus struct {
	Metrics    bool
	URL        string
	User       string
	Password   string
	Timeout    time.Duration
	StrictLint bool
}

func AddPrometheusFlags(fs *pflag.FlagSet, prometheus *Prometheus) {
	fs.BoolVar(
		&prometheus.Metrics,
		"prometheus-metrics",
		false,
		"send test suite run results to a prometheus remote write endpoint.",
	)
	fs.StringVar(
		&prometheus.URL,
		"prometheus-url",
		"",
		"prometheus remote write URL. If not set PROMETHEUS_URL environment variable is used.",
	)
	fs.StringVar(
		&prometheus.User,
		"prometheus-user",
		"",
		"prometheus remote write user. If not set PROMETHEUS_USER environment variable is used.",
	)
	fs.StringVar(
		&prometheus.Password,
		"prometheus-password",
		"",
		"prometheus remote write password. If not set PROMETHEUS_PASSWORD environment variable is used.",
	)
	fs.DurationVar(
		&prometheus.Timeout,
		"prometheus-timeout",
		0,
		"prometheus remote write timeout. If not set PROMETHEUS_TIMEOUT environment variable is used.",
	)
	fs.BoolVar(
		&prometheus.StrictLint,
		"prometheus-strict-lint",
		false,
		"strict lint prometheus metrics. If set to true, will fail if metric does not pass linting",
	)
}

func (config BenchConfig) BuildTestExecutor(
	log *slog.Logger,
	testExecutor string,
) (executor.TestExecutor, error) {
	var executor executor.TestExecutor

	switch config.Test.Executor {
	case "go":
		executor = gotest.NewGoExecutor(
			log,
			gotest.GoExecutorOptions{
				GoArgs:   config.Go.GoArgs,
				Packages: config.Go.Packages,
				TestArgs: config.Go.TestArgs,
				Retries:  config.Go.Retries,
			},
		)
	case "k6":
		executor = k6.NewK6TestExecutor(
			log,
			k6.K6ExecutorOptions{
				Verbose:        config.Test.Verbose,
				CloudOutput:    config.K6.CloudOutput,
				CloudToken:     config.K6.CloudToken,
				CloudProjectID: config.K6.CloudProjectId,
			},
		)
	case "playwright":
		executor = playwright.NewPlaywrightTestExecutor(
			log,
			config.Test.Verbose,
			config.Playwright.PrepareCmd,
			config.Playwright.ExecuteCmd,
		)
	default:
		return nil, fmt.Errorf("invalid test executor %q", testExecutor)
	}

	return executor, nil
}

func (config *BenchConfig) BuildTestSuite(log *slog.Logger) (*executor.TestSuite, error) {
	testSuiteRevision := config.TestSuite.Revision
	if config.TestSuite.Repo != "" {
		log.Info("checking out test suite", "repository", config.TestSuite.Repo)

		compiler := compile.NewTestCompiler(
			log,
			config.TestSuite.BaseDir,
			config.TestSuite.Repo,
			config.TestSuite.RepoDirs,
			config.TestSuite.RepoToken,
			config.TestSuite.Revision,
			[]string{},
		)

		revisionHash, err := compiler.CompileTestSuite(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("checking out test suite: %w", err)
		}

		if testSuiteRevision == "" {
			testSuiteRevision = revisionHash
		}
	}

	// if the test suite name was not given, use repo name (if Any) and the last element of the test suite path
	if config.TestSuite.Name == "" {
		name := strings.TrimSuffix(path.Base(config.TestSuite.Path), path.Ext(config.TestSuite.Path))
		if config.TestSuite.Repo != "" {
			repoURL, _ := url.Parse(config.TestSuite.Repo)
			name, _ = strings.CutPrefix(filepath.Join(repoURL.Path, name), "/")
		}
		config.TestSuite.Name = name
	}

	return &executor.TestSuite{
		Name:     config.TestSuite.Name,
		BaseDir:  config.TestSuite.BaseDir,
		Path:     config.TestSuite.Path,
		Revision: testSuiteRevision,
	}, nil
}

func (benchConfig *BenchConfig) BuildSuiteRun() (executor.SuiteRun, error) {
	grafanaSlug := ""
	if benchConfig.Grafana.Url != "" {
		// in case of error, slug it will be empty
		grafanaSlug, _ = grafana.Slug(benchConfig.Grafana.Url)
	}

	// get grafana version if not provided
	grafanaVersion := benchConfig.Grafana.Version
	if grafanaVersion == "" {
		if benchConfig.Grafana.Url == "" || benchConfig.Grafana.AdminUser == "" || benchConfig.Grafana.AdminPassword == "" {
			return executor.SuiteRun{}, fmt.Errorf("grafana admin user and password are needed to get grafana version")
		}

		grafanaInstance, err := grafana.NewInstance(
			benchConfig.Grafana.Url,
			benchConfig.Grafana.AdminUser,
			benchConfig.Grafana.AdminPassword,
		)
		if err != nil {
			return executor.SuiteRun{}, fmt.Errorf("failed to create grafana instance: %w", err)
		}
		grafanaVersion, err = grafanaInstance.GetGrafanaBuildVersion()
		if err != nil {
			return executor.SuiteRun{}, fmt.Errorf("failed to get grafana version: %w", err)
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

	return executor.SuiteRun{
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
	}, nil
}

func (config *BenchConfig) BuildReporter() (reporter.SuiteRunReporter, error) {
	// chain of test reporters
	reporters := []reporter.SuiteRunReporter{}

	// create test reporter
	var suiteReporter reporter.SuiteRunReporter

	// FIXME: this is a quick fix for the missing service attribute
	// There's no way to get the attributes set in the runner's logger to be used
	// in the reporter logger.
	logAttrs := []any{"service", "bench"}
	switch config.Report.Output {
	case "json":
		suiteReporter, _ = reporter.NewLogReporter(reporter.JSONLog, logAttrs)
	case "log":
		suiteReporter, _ = reporter.NewLogReporter(reporter.TextLog, logAttrs)
	case "text":
		suiteReporter = reporter.NewTextReporter(os.Stdout)
	default:
		return nil, fmt.Errorf("invalid report format %q", config.Report.Output)
	}
	reporters = append(reporters, suiteReporter)

	if config.Slack.Notifications {
		if config.Slack.Token == "" {
			return nil, fmt.Errorf("no slack token provided")
		}

		codeownersMap := config.Slack.CodeownersMap
		if !filepath.IsAbs(codeownersMap) {
			codeownersMap = filepath.Join(config.TestSuite.BaseDir, codeownersMap)
		}
		notifier, err := notifier.NewSlackNotifier(notifier.SlackNotifierOptions{
			Token:        config.Slack.Token,
			MappingFile:  codeownersMap,
			DashboardURL: config.SuiteRun.DashboardURL,
		})
		if err != nil {
			return nil, fmt.Errorf("creating slack notifier: %w", err)
		}

		notificationReporter, err := reporter.NewNotificationReporter(
			config.TestSuite.BaseDir,
			notifier,
			reporter.NotifyPassing(config.Slack.NotifyPassing),
		)
		if err != nil {
			return nil, fmt.Errorf("creating notification reporter: %w", err)
		}

		reporters = append(reporters, notificationReporter)
	}

	if config.Prometheus.Metrics {
		prometheusReporter := reporter.NewPrometheusReporter(reporter.PrometheusConfig{
			URL:      config.Prometheus.URL,
			User:     config.Prometheus.User,
			Password: config.Prometheus.Password,
			Timeout:  config.Prometheus.Timeout,
			Prefix:   config.SuiteRun.MetricsPrefix,
		})

		reporters = append(reporters, prometheusReporter)
	}

	return reporter.NewChainReporter(reporters...), nil
}

func (config *BenchConfig) GetGrafanaInstance(log *slog.Logger) (grafana.GrafanaInstance, string, error) {
	grafanaInstance, err := grafana.NewInstance(
		config.Grafana.Url,
		config.Grafana.AdminUser,
		config.Grafana.AdminPassword,
		grafana.WithTimeout(config.Grafana.Timeout),
	)
	if err != nil {
		return nil, "", err
	}

	log.Info("Waiting for grafana server...", "address", grafanaInstance.Url())

	err = grafanaInstance.WaitForLiveGrafana(context.TODO())
	if err != nil {
		return nil, "", fmt.Errorf("checking Grafana is Live... %w", err)
	}
	log.Debug("Grafana server is ready!")

	grafanaVersion, err := grafanaInstance.GetGrafanaBuildVersion()
	if err != nil {
		return nil, "", fmt.Errorf("getting grafana version %w", err)
	}

	return grafanaInstance, grafanaVersion, nil
}

func (config *BenchConfig) GetRunMetrics(log *slog.Logger) ([]metrics.Metric, error) {
	metricList := []metrics.Metric{}
	for _, metricString := range config.SuiteRun.Metrics {
		metric, err := metrics.ParseMetric(metricString)
		if err != nil {
			return nil, err
		}
		metricList = append(metricList, metric)
	}

	if config.SuiteRun.MetricsFile != "" {
		metricsFromFile, err := metrics.ParseMetricsFile(config.SuiteRun.MetricsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to parse metrics from file %s: %w", config.SuiteRun.MetricsFile, err)
		}
		metricList = append(metricList, metricsFromFile...)

		err = metrics.LintMetrics(log, metricList, config.Prometheus.StrictLint)
		if err != nil {
			return []metrics.Metric{}, err
		}

	}

	return metricList, nil
}
