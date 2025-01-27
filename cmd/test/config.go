// Package config defines the test configuration
package test

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
	"github.com/grafana/grafana-bench/pkg/executor/k6"
	"github.com/grafana/grafana-bench/pkg/executor/playwright"
	"github.com/grafana/grafana-bench/pkg/notifier"
	"github.com/grafana/grafana-bench/pkg/reporter"
	"github.com/grafana/grafana-bench/pkg/revision"
	"github.com/grafana/grafana-bench/pkg/runner"
	"github.com/grafana/grafana-bench/pkg/utils/env"
)

type BenchConfig struct {
	// FIXME: moved there because is needed by the slack notifications
	// for the codeowners mapping.
	BaseDir            string
	BenchRevision      string
	Type               string
	Trigger            string
	EnvVars            map[string]string
	LogLevel           string
	ReportFormat       string
	Verbose            bool
	SlackNotifications bool
	NotifyPassing      bool
	DashboardURL       string
	Grafana            GrafanaConfig
	K6                 K6Config
	PW                 PWConfig
	Slack              SlackNotifierConfig
}

type GrafanaConfig struct {
	Url           string
	AdminUser     string
	AdminPassword string
	Timeout       time.Duration
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
	// FIXME: doing here because we don't have any other place to do it
	// needed after BaseDir was moved to BenchConfig
	if c.BaseDir == "" {
		c.BaseDir, _ = os.Getwd()
	}

	if c.BenchRevision == "" {
		c.BenchRevision = env.EnvOrDefault("BENCH_REVISION", revision.BenchRevision())
	}

	// Grafana
	c.Grafana.Url = env.EnvOrDefault("GRAFANA_URL", c.Grafana.Url)
	c.Grafana.AdminUser = env.EnvOrDefault("GRAFANA_ADMIN_USER", c.Grafana.AdminUser)
	c.Grafana.AdminPassword = env.EnvOrDefault("GRAFANA_ADMIN_PASSWORD", c.Grafana.AdminPassword)

	// k6 config
	c.K6.CloudToken = env.EnvOrDefault("K6_CLOUD_TOKEN", c.K6.CloudToken)
	c.K6.CloudProjectId = env.EnvOrDefault("K6_CLOUD_PROJECT_ID", c.K6.CloudProjectId)

	c.Slack.Token = env.EnvOrDefault("SLACK_TOKEN", c.Slack.Token)
}

func (config BenchConfig) BuildTestRunner(log *slog.Logger, testExecutor string) (*runner.TestRunner, error) {
	grafanaInstance, grafanaVersion, err := getGrafanaInstance(
		log,
		config.Grafana,
	)
	if err != nil {
		return nil, err
	}

	logAttrs := []any{
		"testTrigger", config.Trigger,
		"testExecutor", testExecutor,
		"benchRevision", config.BenchRevision,
		"grafanaUrl", grafanaInstance.Hostname(),
		"grafanaSlug", grafanaInstance.Slug(),
		"grafanaVersion", grafanaVersion,
	}

	runnerLog := log.With(logAttrs...)

	var executor executor.TestExecutor
	if testExecutor == "k6" {
		executor = k6.NewK6TestExecutor(
			runnerLog,
			k6.K6ExecutorOptions{
				Verbose:        config.Verbose,
				CloudOutput:    config.K6.CloudOutput,
				CloudToken:     config.K6.CloudToken,
				CloudProjectID: config.K6.CloudProjectId,
			},
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
		// FIXME: this is a quick fix for the missing service attribute
		// There's no way to get the attributes set in the runner's logger to be used
		// in the reporter logger.
		reporterAttrs := append(logAttrs, "service", "bench")
		suiteReporter = reporter.NewLogReporter(reporterAttrs)
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

		codeownersMap := config.Slack.CodeownersMap
		if !filepath.IsAbs(codeownersMap) {
			codeownersMap = filepath.Join(config.BaseDir, codeownersMap)
		}
		notifier, err := notifier.NewSlackNotifier(notifier.SlackNotifierOptions{
			Token:        config.Slack.Token,
			MappingFile:  codeownersMap,
			DashboardURL: config.DashboardURL,
		})
		if err != nil {
			return nil, fmt.Errorf("creating slack notifier: %w", err)
		}

		notificationReporter, err := reporter.NewNotificationReporter(
			config.BaseDir,
			notifier,
			reporter.NotifyPassing(config.NotifyPassing),
		)
		if err != nil {
			return nil, fmt.Errorf("creating notification reporter: %w", err)
		}

		reporters = append(reporters, notificationReporter)
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

func (config *TestSuiteConfig) BuildTestSuite(log *slog.Logger, baseDir string) (*executor.TestSuite, error) {
	testSuiteRevision := config.Revision
	if config.Repo != "" {
		log.Info("checking out test suite", "repository", config.Repo)

		compiler := compile.NewTestCompiler(
			log,
			baseDir,
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

	// if the test suite name was not given, use repo name (if Any) and the last element of the test suite path
	if config.Name == "" {
		name := strings.TrimSuffix(path.Base(config.Path), path.Ext(config.Path))
		if config.Repo != "" {
			repoURL, _ := url.Parse(config.Repo)
			name, _ = strings.CutPrefix(filepath.Join(repoURL.Path, name), "/")
		}
		config.Name = name
	}

	return &executor.TestSuite{
		Name:     config.Name,
		BaseDir:  baseDir,
		Path:     config.Path,
		Revision: testSuiteRevision,
	}, nil
}
