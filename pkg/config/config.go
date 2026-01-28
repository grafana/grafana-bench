// Package config defines the bench configuration
package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/pkg/compile"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/executor/gobench"
	"github.com/grafana/grafana-bench/pkg/executor/gotest"
	"github.com/grafana/grafana-bench/pkg/executor/k6"
	"github.com/grafana/grafana-bench/pkg/executor/playwright"
	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/metrics"
	"github.com/grafana/grafana-bench/pkg/notifier"
	"github.com/grafana/grafana-bench/pkg/reporter"
	"github.com/grafana/grafana-bench/pkg/revision"
	"github.com/grafana/grafana-bench/pkg/service"
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
	Service    ServiceConfig
	Go         GoTestConfig
	GoBench    GoBenchConfig
	K6         K6Config
	Playwright PWConfig
	Slack      SlackNotifierConfig
	Prometheus Prometheus
	Git        GitConfig
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

type ServiceConfig struct {
	Name         string
	Version      string
	Url          string
	Timeout      time.Duration
	FetchVersion string // Optional credentials for fetching Grafana version (user:password format)
	HealthCheck  bool   // Whether to perform health check before running tests
}

func AddServiceFlags(fs *pflag.FlagSet, config *ServiceConfig) {
	// Service identifier
	fs.StringVar(
		&config.Name,
		"service",
		"",
		"REQUIRED. Name of the service being tested (e.g., 'grafana', 'loki', 'tempo', 'datasources'). Used for identifying which service the test results belong to in logs and metrics.",
	)

	// Generic service flags
	fs.StringVar(
		&config.Url,
		"service-url",
		"http://localhost:3000",
		"URL to the service being tested. Overridden by the SERVICE_URL environment variable (default http://localhost:3000)",
	)
	fs.DurationVar(
		&config.Timeout,
		"service-timeout",
		grafana.DefaultGrafanaTimeout,
		"timeout for waiting for the service to be live",
	)
	fs.StringVar(
		&config.Version,
		"service-version",
		"",
		"REQUIRED. Version of the service being tested (e.g., '11.0.0', '2.9.0'). Overridden by the SERVICE_VERSION environment variable.",
	)
	fs.BoolVar(
		&config.HealthCheck,
		"service-health-check",
		false,
		"Perform a TCP health check on the service before running tests. Uses --service-url and --service-timeout.",
	)

	// Grafana-specific convenience flag for fetching version
	fs.StringVar(
		&config.FetchVersion,
		"fetch-grafana-version",
		"",
		"Optional: Fetch Grafana version from API using provided credentials in 'user:password' format."+
			"\nMutually exclusive with --service-version. Overridden by FETCH_GRAFANA_VERSION environment variable."+
			"\nExample: --fetch-grafana-version=admin:admin",
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
		"pw-prepare",
		"",
		"commands used to install dependencies for the test suite eg: \"npm install\"."+
			"\nMultiple commands can be specified by separating with ';'.",
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

type GoBenchConfig struct {
	GoArgs       []string
	BenchArgs    []string
	Packages     []string
	BenchPattern string
	BenchTime    string
	BenchMem     bool
	Count        int
}

func AddGoBenchExecutorFlags(fs *pflag.FlagSet, config *GoBenchConfig) {
	fs.StringArrayVar(
		&config.Packages,
		"gobench-packages",
		nil,
		"Package patterns for benchmarks (e.g., './...', './pkg/...'). If not specified, defaults to './...'",
	)
	fs.StringVar(
		&config.BenchPattern,
		"gobench-pattern",
		".",
		"Benchmark name pattern (regex). Use '.' to run all benchmarks",
	)
	fs.StringVar(
		&config.BenchTime,
		"gobench-time",
		"",
		"Benchmark duration (e.g., '10s', '100x'). If not set, uses Go's default",
	)
	fs.BoolVar(
		&config.BenchMem,
		"gobench-mem",
		true,
		"Enable memory statistics (B/op, allocs/op)",
	)
	fs.IntVar(
		&config.Count,
		"gobench-count",
		1,
		"Number of times to run each benchmark",
	)
	fs.StringArrayVar(
		&config.GoArgs,
		"gobench-args",
		nil,
		"Additional arguments passed to 'go test' (e.g., '-tags=integration')",
	)
	fs.StringArrayVar(
		&config.BenchArgs,
		"gobench-bench-args",
		nil,
		"Arguments passed to benchmarks via -args",
	)
}

type SuiteRunConfig struct {
	RunStage      string
	Id            string
	DashboardURL  string
	Metrics       []string
	MetricsPrefix string
	MetricsFile   string
	Attributes    []string
}

func AddSuiteRunFlags(fs *pflag.FlagSet, config *SuiteRunConfig) {
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
		&config.RunStage,
		"run-stage",
		"local",
		"the stage of CI the suite was executed. For example, 'local', 'ci', 'rrc'.",
	)
	fs.StringArrayVar(
		&config.Metrics,
		"run-metric",
		nil,
		"test suite run custom metrics. Format: name{label=label-value,..}=value. The value must be a valid float number.",
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
	fs.StringArrayVar(
		&config.Attributes,
		"run-attribute",
		nil,
		"adds custom attributes to a suite run. Good for descriptive information. Format: --run-attribute=\"key=value,key=value\". Attributes with no value will be skipped. You can either use the comma separated format shown here or call --run-attribute multiple times to add additional attributes",
	)
}

type ReportConfig struct {
	Output string
	Input  string
}

func AddReportOutputFlags(fs *pflag.FlagSet, report *ReportConfig) {
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
	Verbose    bool
	Type       string
	Executor   string
	Env        map[string]string // Parsed env vars (for backward compat with config files)
	EnvRaw     []string          // Raw --test-env flags for passthrough support
	EnvVarsRaw []string          // Deprecated: raw --test-env-vars flags
}

func AddTestFlags(fs *pflag.FlagSet, test *TestConfig) {
	AddTestVeboseFlag(fs, test)
	AddTestEnvFlags(fs, test)
	AddTestTypeFlag(fs, test)
	AddTestRunnerFlag(fs, test)
}

func AddTestEnvFlags(fs *pflag.FlagSet, test *TestConfig) {
	fs.StringSliceVar(
		&test.EnvRaw,
		"test-env",
		nil,
		"environment variables passed to the test execution. "+
			"Use 'KEY=VALUE' to set explicitly, or 'KEY' to pass through from environment (secure for credentials).",
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
		"test runner. Allowed values: 'k6', 'playwright', 'go', 'gobench'",
	)
}

func AddTestVeboseFlag(fs *pflag.FlagSet, test *TestConfig) {
	fs.BoolVar(
		&test.Verbose,
		"test-verbose",
		false,
		"show test output",
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
		"suite-name",
		"",
		"[REQUIRED] Test suite name used for identifying and labeling test results in logs and metrics."+
			"\nIf not specified, SUITE_NAME environment variable is used."+
			"\nExample: 'grafana-bench/go-tests' or 'my-project/smoke-tests'",
	)
}

func AddSuitePathFlags(fs *pflag.FlagSet, config *TestSuiteConfig) {
	fs.StringVar(
		&config.BaseDir,
		"suite-base",
		".",
		"base directory for searching test suites. Defaults to current directory"+
			"\nIf specified, it is prefixed to the --suite-path.",
	)
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
		"suite-repo-url",
		"",
		"url to the repository to get the test suite from. If not set SUITE_REPO_URL environment variable is used."+
			"\nIf specified, the repo will be checkout into the --suite-base directory."+
			"\nIf --suite-revision is specified, that revision will be checkout."+
			"\nOtherwise the default branch will be checkout",
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
		"suite-repo-dirs",
		nil,
		"Directories to checkout from test suite repo. If omitted, all folders will be checkout",
	)
}

func AddSuiteRevisionFlag(fs *pflag.FlagSet, config *TestSuiteConfig) {
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
	AddSlackNotificationsFlag(fs, config)
	AddSlackNotifyPassingFlag(fs, config)
	AddSlackToken(fs, config)
	AddSlackCodeownersMapFlag(fs, config)
}

func AddSlackNotificationsFlag(fs *pflag.FlagSet, config *SlackNotifierConfig) {
	fs.BoolVar(
		&config.Notifications,
		"slack-notifications",
		false,
		"send notifications to slack. Requires setting the --slack-token option or the SLACK_TOKEN environment variable.",
	)
}

func AddSlackNotifyPassingFlag(fs *pflag.FlagSet, config *SlackNotifierConfig) {
	fs.BoolVar(
		&config.NotifyPassing,
		"slack-passing",
		false,
		"send notifications for passing test suites. By default only not passing test suites are notified",
	)
}

func AddSlackToken(fs *pflag.FlagSet, config *SlackNotifierConfig) {
	fs.StringVar(
		&config.Token,
		"slack-token",
		"",
		"slack token used for sending notifications. If not defined SLACK_TOKEN environment variable is used."+
			"\nThe token requires chat:write and channels:read scopes",
	)
}
func AddSlackCodeownersMapFlag(fs *pflag.FlagSet, config *SlackNotifierConfig) {
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

type GitConfig struct {
	Driver string
}

func AddGitFlags(fs *pflag.FlagSet, git *GitConfig) {
	fs.StringVar(
		&git.Driver,
		"git-driver",
		"nanogit",
		"git driver used for downloading the test suite repo ('nanogit', 'gogit').",
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
	case "gobench":
		executor = gobench.NewGoBenchExecutor(
			log,
			gobench.GoBenchExecutorOptions{
				GoArgs:       config.GoBench.GoArgs,
				BenchArgs:    config.GoBench.BenchArgs,
				Packages:     config.GoBench.Packages,
				BenchPattern: config.GoBench.BenchPattern,
				BenchTime:    config.GoBench.BenchTime,
				BenchMem:     config.GoBench.BenchMem,
				Count:        config.GoBench.Count,
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
			config.Git.Driver,
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

	// Validate required test suite name
	if config.TestSuite.Name == "" {
		return nil, fmt.Errorf("--suite-name is required\n" +
			"The suite name identifies your tests in logs and metrics.\n" +
			"Use format: <project>/<test-type>\n" +
			"Examples:\n" +
			"  --suite-name grafana-bench/go-tests\n" +
			"  --suite-name my-plugin/e2e-tests\n" +
			"  --suite-name api-service/benchmarks")
	}

	return &executor.TestSuite{
		Name:     config.TestSuite.Name,
		BaseDir:  config.TestSuite.BaseDir,
		Path:     config.TestSuite.Path,
		Revision: testSuiteRevision,
	}, nil
}

func (benchConfig *BenchConfig) BuildSuiteRun(log *slog.Logger) (executor.SuiteRun, error) {
	// Only set service URL for executors that test service endpoints (k6, playwright)
	// For go/gobench, this doesn't make sense since they're testing code directly
	serviceURL := ""
	if benchConfig.Test.Executor == "k6" || benchConfig.Test.Executor == "playwright" {
		serviceURL = benchConfig.Service.Url
	}

	// Validate required service field
	if benchConfig.Service.Name == "" {
		return executor.SuiteRun{}, fmt.Errorf("--service is required: specify the name of the service being tested (e.g., 'grafana', 'loki', 'tempo')")
	}

	// Handle service version - either explicit or fetched from Grafana API
	serviceVersion := benchConfig.Service.Version

	// Check for mutually exclusive version options
	if serviceVersion != "" && benchConfig.Service.FetchVersion != "" {
		return executor.SuiteRun{}, fmt.Errorf("--service-version and --fetch-grafana-version are mutually exclusive: use only one")
	}

	// Fetch version from Grafana API if requested
	if benchConfig.Service.FetchVersion != "" {
		// Parse user:password format
		parts := strings.SplitN(benchConfig.Service.FetchVersion, ":", 2)
		if len(parts) != 2 {
			return executor.SuiteRun{}, fmt.Errorf("--fetch-grafana-version must be in 'user:password' format (e.g., 'admin:admin')")
		}
		username, password := parts[0], parts[1]

		if username == "" || password == "" {
			return executor.SuiteRun{}, fmt.Errorf("--fetch-grafana-version: both username and password are required")
		}

		if benchConfig.Service.Url == "" {
			return executor.SuiteRun{}, fmt.Errorf("--service-url is required when using --fetch-grafana-version")
		}

		// Wait for service to be live before attempting to fetch version
		log.Info("waiting for service to be ready...", "url", benchConfig.Service.Url, "timeout", benchConfig.Service.Timeout)
		healthCheckOpts := service.HealthCheckOptions{
			Timeout: benchConfig.Service.Timeout,
			Backoff: 1 * time.Second,
		}
		err := service.WaitForServiceLive(context.TODO(), benchConfig.Service.Url, healthCheckOpts)
		if err != nil {
			return executor.SuiteRun{}, fmt.Errorf("service health check failed: %w", err)
		}
		log.Info("service is ready")

		grafanaInstance, err := grafana.NewInstance(
			benchConfig.Service.Url,
			username,
			password,
		)
		if err != nil {
			return executor.SuiteRun{}, fmt.Errorf("failed to create grafana instance for version fetching: %w", err)
		}

		serviceVersion, err = grafanaInstance.GetGrafanaBuildVersion()
		if err != nil {
			return executor.SuiteRun{}, fmt.Errorf("failed to fetch grafana version: %w", err)
		}

		log.Info("fetched grafana version from API", "version", serviceVersion)
	}

	// Validate that version is provided
	if serviceVersion == "" {
		return executor.SuiteRun{}, fmt.Errorf("--service-version is required: specify the version of the service being tested (e.g., '11.0.0'), or use --fetch-grafana-version for Grafana")
	}

	// get attributes of this test suite run using the test suite information from the config
	runId := benchConfig.SuiteRun.Id
	if runId == "" {
		runId = id.Run(
			benchConfig.SuiteRun.RunStage,
			benchConfig.TestSuite.Name,
			time.Now(),
		)
	}

	attributes, err := parseAttributes(benchConfig.SuiteRun.Attributes, log)
	if err != nil {
		return executor.SuiteRun{}, err
	}

	return executor.SuiteRun{
		Id:             runId,
		RunStage:       benchConfig.SuiteRun.RunStage,
		Service:        benchConfig.Service.Name,
		TestExecutor:   benchConfig.Report.Input,
		Attributes:     attributes,
		BenchRevision:  benchConfig.Revision,
		ServiceURL:     serviceURL,
		ServiceVersion: serviceVersion,
	}, nil
}

func (config *BenchConfig) BuildReporter() (reporter.SuiteRunReporter, error) {
	// chain of test reporters
	reporters := []reporter.SuiteRunReporter{}

	// create test reporter
	var suiteReporter reporter.SuiteRunReporter

	// Set tool=bench to identify that bench is running the tests
	// Add service attribute to identify what service is being tested
	logAttrs := []any{"tool", "bench", "service", config.Service.Name}
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
		// Validate required Prometheus configuration
		if config.Prometheus.URL == "" {
			return nil, fmt.Errorf("--prometheus-metrics requires PROMETHEUS_URL environment variable or --prometheus-url flag to be set")
		}
		if config.Prometheus.User == "" {
			return nil, fmt.Errorf("--prometheus-metrics requires PROMETHEUS_USER environment variable or --prometheus-user flag to be set")
		}
		if config.Prometheus.Password == "" {
			return nil, fmt.Errorf("--prometheus-metrics requires PROMETHEUS_PASSWORD environment variable or --prometheus-password flag to be set")
		}

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

// parseAttributes parses the cobra stringArrayVar into a map[string]string of attributes
// for ux, we allow a user to add multiple attributes in a single stringArrayVar
// by using the format --run-attribute="key=value,key=value". If there are overlapping keys
// the last one takes precedence
func parseAttributes(attributes []string, log *slog.Logger) (map[string]string, error) {
	attributeList := map[string]string{}

	for _, attributeString := range attributes {
		if strings.TrimSpace(attributeString) == "" {
			continue
		}

		attrs := strings.Split(attributeString, ",")
		for _, attr := range attrs {
			attr = strings.TrimSpace(attr)
			if attr == "" {
				continue
			}

			kv := strings.SplitN(attr, "=", 2)
			if len(kv) < 2 {
				return nil, fmt.Errorf("invalid attribute format %q: expected 'key=value'", attr)
			}

			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			if key == "" {
				return nil, fmt.Errorf("empty key in attribute %q", attr)
			}

			if value == "" {
				log.Warn("parsing run attributes: skipping attribute with empty value", "key", key)
				continue
			}

			attributeList[key] = value
		}
	}

	return attributeList, nil
}
