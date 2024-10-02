// Package config defines the test configuration
package test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/pkg/compile"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/executor/k6"
	"github.com/grafana/grafana-bench/pkg/executor/playwright"
	"github.com/grafana/grafana-bench/pkg/notifier"
	"github.com/grafana/grafana-bench/pkg/reporter"
	"github.com/grafana/grafana-bench/pkg/revision"
	"github.com/grafana/grafana-bench/pkg/runner"
	"github.com/grafana/grafana-bench/pkg/utils/env"
)

type BenchConfig struct {
	BenchRevision      string
	Type               string
	Trigger            string
	EnvVars            map[string]string
	LogLevel           string
	ReportFormat       string
	Verbose            bool
	SlackNotifications bool
	DashboardURL       string
	Grafana            GrafanaConfig
	K6                 K6Config
	PW                 PWConfig
	Slack              SlackNotifierConfig
}

type GrafanaConfig struct {
	Url      string
	UserName string
	Password string
	Timeout  time.Duration
}

type K6Config struct {
	CloudToken     string
	CloudProjectId string
	CloudOutput    bool
}

type PWConfig struct {
	PrepareCmd string
	ExecuteCmd string
}

type TestSuiteConfig struct {
	GitToken     string
	Name         string
	Repo         string
	RepoToken    string
	RepoDirs     []string
	Path         string
	BaseDir      string
	Revision     string
	TestExecutor string
}

type SlackNotifierConfig struct {
	CodeownersMap string
	Token         string
}

// MergeEnv updates the config by overriding some fields with environment variables.
// Any environment variables that are not set will not override the config fields.
func (c *BenchConfig) MergeEnv() {
	if c.BenchRevision == "" {
		c.BenchRevision = env.EnvOrDefault("BENCH_REVISION", revision.BenchRevision())
	}

	// Grafana
	c.Grafana.Url = env.EnvOrDefault("GRAFANA_URL", c.Grafana.Url)
	c.Grafana.UserName = env.EnvOrDefault("GRAFANA_USER", c.Grafana.UserName)
	c.Grafana.Password = env.EnvOrDefault("GRAFANA_PASSWORD", c.Grafana.Password)

	// k6 config
	c.K6.CloudToken = env.EnvOrDefault("K6_CLOUD_TOKEN", c.K6.CloudToken)
	c.K6.CloudProjectId = env.EnvOrDefault("K6_CLOUD_PROJECT_ID", c.K6.CloudProjectId)

	c.Slack.Token = env.EnvOrDefault("SLACK_TOKEN", c.Slack.Token)
}

func (config BenchConfig) BuildTestRunner(log *slog.Logger, testExecutor string) (*runner.TestRunner, error) {
	grafanaInstance, grafanaVersion, err := getGrafanaInstance(
		log,
		config.Grafana.Url,
		config.Grafana.UserName,
		config.Grafana.Password,
		config.Grafana.Timeout,
	)
	if err != nil {
		return nil, err
	}

	runnerLog := log.With(
		"testTrigger", config.Trigger,
		"testExecutor", testExecutor,
		"benchRevision", config.BenchRevision,
		"grafanaUrl", grafanaInstance.Hostname(),
		"grafanaSlug", grafanaInstance.Slug(),
		"grafanaVersion", grafanaVersion,
	)

	var executor executor.TestExecutor
	if testExecutor == "k6" {
		executor = k6.NewK6TestExecutor(
			runnerLog,
			config.Verbose,
			config.K6.CloudOutput,
			config.K6.CloudToken,
			config.K6.CloudProjectId,
		)
	}

	if testExecutor == "playwright" {
		executor = playwright.NewPlaywrightTestExecutor(
			runnerLog,
			config.Verbose,
			config.PW.PrepareCmd,
			config.PW.ExecuteCmd,
		)
	}

	// chain of test reporters
	reporters := []reporter.SuiteRunReporter{}

	// create test reporter
	var suiteReporter reporter.SuiteRunReporter
	switch config.ReportFormat {
	case "log":
		suiteReporter = reporter.NewLogReporter(runnerLog)
	case "text":
		suiteReporter = reporter.NewTextReporter(os.Stdout)
	default:
		return nil, fmt.Errorf("invalid report format %q", config.ReportFormat)
	}
	reporters = append(reporters, suiteReporter)

	if config.SlackNotifications {
		if config.Slack.Token == "" {
			return nil, fmt.Errorf("no slack token provided")
		}

		notifier, err := notifier.NewSlackNotifier(notifier.SlackNotifierOptions{
			Token:        config.Slack.Token,
			MappingFile:  config.Slack.CodeownersMap,
			DashboardURL: config.DashboardURL,
		})

		if err != nil {
			return nil, fmt.Errorf("creating slack notifier: %w", err)
		}

		reporters = append(reporters, reporter.NewNotificationReporter(notifier, reporter.NotifyAll))
	}

	return runner.NewTestRunner(
		runnerLog,
		config.Trigger,
		grafanaInstance,
		grafanaVersion,
		config.BenchRevision,
		config.DashboardURL,
		executor,
		reporter.NewChainReporter(reporters...),
	), nil
}

// MergeEnv updates the config by overriding some fields with environment variables.
// Any environment variables that are not set will not override the config fields.
func (c *TestSuiteConfig) MergeEnv() {
	c.Revision = env.EnvOrDefault("TEST_SUITE_REVISION", c.Revision)
	c.Name = env.EnvOrDefault("TEST_SUITE_NAME", c.Name)
	c.RepoToken = env.EnvOrDefault("TEST_SUITE_REPO_TOKEN", c.RepoToken)
}

func (config *TestSuiteConfig) BuildTestSuite(log *slog.Logger) (*executor.TestSuite, error) {
	var err error

	// if the name of the test suite was not given, use the last element of the test suit path as name
	if config.Name == "" {
		config.Name = strings.TrimSuffix(path.Base(config.Path), path.Ext(config.Path))
	}

	if config.BaseDir == "" {
		config.BaseDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting work directory %w", err)
		}
	}

	testSuiteRevision := config.Revision
	if config.Repo != "" {
		log.Info("checking out test suite", "repository", config.Repo)

		compiler := compile.NewTestCompiler(
			log,
			config.BaseDir,
			config.Repo,
			config.RepoDirs,
			config.RepoToken,
			config.Revision,
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

	return &executor.TestSuite{
		Name:     config.Name,
		BaseDir:  config.BaseDir,
		Path:     config.Path,
		Revision: testSuiteRevision,
	}, nil
}
