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
	"github.com/grafana/grafana-bench/pkg/executor/k6"
	"github.com/grafana/grafana-bench/pkg/executor/playwright"
	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/notifier"
	"github.com/grafana/grafana-bench/pkg/reporter"
)

type BenchConfig struct {
	// FIXME: moved there because is needed by the slack notifications
	// for the codeowners mapping.
	BaseDir       string
	BenchRevision string
	TestSuite     TestSuiteConfig
	Test          TestConfig
	Report        ReportConfig
	SuiteRun      SuiteRunConfig
	LogLevel      string
	Verbose       bool
	Grafana       GrafanaConfig
	K6            K6Config
	PW            PWConfig
	Slack         SlackNotifierConfig
}

type GrafanaConfig struct {
	Version       string
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

type SuiteRunConfig struct {
	Trigger       string
	Id            string
	DashboardURL  string
	Metrics       map[string]string
	MetricsPrefix string
}

type ReportConfig struct {
	Format string
}

type TestConfig struct {
	Type   string
	Runner string
	Env    map[string]string
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
	Notifications bool
	NotifyPassing bool
}

func (config BenchConfig) BuildTestExecutor(
	log *slog.Logger,
	testExecutor string,
	grafanaInstance grafana.GrafanaInstance,
	grafanaVersion string,
) (executor.TestExecutor, error) {
	var executor executor.TestExecutor

	switch config.Test.Runner {
	case "k6":
		executor = k6.NewK6TestExecutor(
			log,
			k6.K6ExecutorOptions{
				Verbose:        config.Verbose,
				CloudOutput:    config.K6.CloudOutput,
				CloudToken:     config.K6.CloudToken,
				CloudProjectID: config.K6.CloudProjectId,
			},
		)
	case "playwright":
		executor = playwright.NewPlaywrightTestExecutor(
			log,
			config.Verbose,
			config.PW.PrepareCmd,
			config.PW.ExecuteCmd,
		)
	default:
		return nil, fmt.Errorf("invalid test executor %q", testExecutor)
	}

	return executor, nil
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

func (config *BenchConfig) BuildReporter() (reporter.SuiteRunReporter, error) {
	// chain of test reporters
	reporters := []reporter.SuiteRunReporter{}

	// create test reporter
	var suiteReporter reporter.SuiteRunReporter

	// FIXME: this is a quick fix for the missing service attribute
	// There's no way to get the attributes set in the runner's logger to be used
	// in the reporter logger.
	logAttrs := []any{"service", "bench"}
	switch config.Report.Format {
	case "log":
		suiteReporter, _ = reporter.NewLogReporter(reporter.TextLog, logAttrs)
	case "json":
		suiteReporter, _ = reporter.NewLogReporter(reporter.JSONLog, logAttrs)
	case "text":
		suiteReporter = reporter.NewTextReporter(os.Stdout)
	default:
		return nil, fmt.Errorf("invalid report format %q", config.Report.Format)
	}
	reporters = append(reporters, suiteReporter)

	if config.Slack.Notifications {
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
			DashboardURL: config.SuiteRun.DashboardURL,
		})
		if err != nil {
			return nil, fmt.Errorf("creating slack notifier: %w", err)
		}

		notificationReporter, err := reporter.NewNotificationReporter(
			config.BaseDir,
			notifier,
			reporter.NotifyPassing(config.Slack.NotifyPassing),
		)
		if err != nil {
			return nil, fmt.Errorf("creating notification reporter: %w", err)
		}

		reporters = append(reporters, notificationReporter)
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
