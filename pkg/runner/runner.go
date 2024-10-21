// Package runner implements the test runners
package runner

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"time"

	"github.com/grafana/grafana-bench/pkg/coverage"
	"github.com/grafana/grafana-bench/pkg/dashboard"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/openapi"
	"github.com/grafana/grafana-bench/pkg/recorder"
	"github.com/grafana/grafana-bench/pkg/reporter"
)

type TestRunner struct {
	Log             *slog.Logger
	Trigger         string
	GrafanaInstance grafana.GrafanaInstance
	GrafanaVersion  string
	BenchRevision   string
	DashboardURL    string
	Executor        executor.TestExecutor
	Reporter        reporter.SuiteRunReporter
	ReportAPICoverage    bool
}

func NewTestRunner(
	log *slog.Logger,
	testTrigger string,
	grafanaInstance grafana.GrafanaInstance,
	grafanaVersion string,
	benchRevision string,
	dashboardURL string,
	executor executor.TestExecutor,
	reporter reporter.SuiteRunReporter,
	reportAPICoverage bool,

) *TestRunner {
	return &TestRunner{
		Log:             log.With("svc", "test-runner"),
		Trigger:         testTrigger,
		GrafanaInstance: grafanaInstance,
		GrafanaVersion:  grafanaVersion,
		BenchRevision:   benchRevision,
		DashboardURL:    dashboardURL,
		Executor:        executor,
		Reporter:        reporter,
		ReportAPICoverage:    reportAPICoverage,
	}
}

func (t *TestRunner) Exec(ctx context.Context, testType TestType, suite executor.TestSuite, testVars map[string]string) error {
	var err error

	// get an unique identification for the run
	runId := t.getRunId(testType)
	t.Log = t.Log.With("runId", runId)

	// get an unique identification for the suite run (used for backward compatibility)
	suiteRunId := t.getSuiteRunId(runId, suite)
	t.Log = t.Log.With("suiteRun", suiteRunId)

	var proxy *recorder.ProxyRecorder
	if t.ReportAPICoverage {
		proxy, err = recorder.NewProxyRecorder(recorder.ProxyOptions{
			Target: t.GrafanaInstance.Address(),
		})
		if err != nil {
			return fmt.Errorf("starting recorder API %v", err)
		}
		testVars["HTTPS_PROXY"] = proxy.ProxyHost()
	}

	// set common test execution variables
	env := map[string]string{
		"TEST_TYPE":           testType.Name(),
		"TEST_SUITE_REVISION": suite.Revision,
		"GRAFANA_URL":         t.GrafanaInstance.Url(),
		"GRAFANA_USERNAME":    t.GrafanaInstance.UserName(),
		"GRAFANA_PASSWORD":    t.GrafanaInstance.Password(),
	}

	maps.Copy(env, testVars)

	suiteRun, err := t.Executor.ExecTestSuite(ctx, suite, env)
	if err != nil {
		return fmt.Errorf("executing test suite: %w", err)
	}

	// TODO: handle error from reporter
	err = t.Reporter.Report(ctx, runId, suiteRunId, suite, suiteRun)
	if err != nil {
		t.Log.Error("reporting test suite run", "error", err)
	}

	var anyFailures = suiteRun.Status != executor.SuitePassed

	if anyFailures {
		dashboardMsg := ""
		if t.DashboardURL != "" {
			// This functionality is ALPHA and may be removed in favor of outputting
			// the suiteRun ID and leaving it up to the user.
			dashboard, err := dashboard.RenderDashboardURL(t.DashboardURL, runId)
			if err != nil {
				t.Log.Error("getting URL dashboard", "error", err)
			} else {
				dashboardMsg = fmt.Sprintf(". See dashboard: %s", dashboard)
			}
		}

		return fmt.Errorf("test suite failed: Too many test failures%s", dashboardMsg)
	}

	if t.ReportAPICoverage {
		recording, err := proxy.GetRecording()
		if err != nil {
			return fmt.Errorf("getting recording %w", err)
		}

		httpbinAPI, err := openapi.FromFile("grafanaV3.json")
		if err != nil {
			return fmt.Errorf("loading HTTPBIN API %w", err)
		}

		analizer, err := coverage.NewAnalizer("/api", "", httpbinAPI)
		if err != nil {
			return fmt.Errorf("loading HTTPBIN API %v", err)
		}

		analizer.Analize(recording)

		report, err := analizer.Coverage("/api")
		if err != nil {
			return fmt.Errorf("getting coverage %v", err)
		}

		// TODO: add CLI options to control print options
		report.Print(coverage.PrintOptions{
			MaxDepth: 0,
			Indent: true,
			SkipUncovered: true,
		}, os.Stdout)
	}

	return nil
}

// returns an unique id for the run
// format: {test type}-{year}{day of year}-{hour}{min}{second}
// Example load-2024123-140035
func (t *TestRunner) getRunId(testType TestType) string {
	now := time.Now().UTC()
	return fmt.Sprintf("%s-%d%d-%d%d%d",
		testType.Name(),
		now.Year(),
		now.YearDay(),
		now.Hour(),
		now.Minute(),
		now.Second(),
	)
}

// returns an unique id for the suite run (DEPRECATED)
// format: {suite name}-{suite-revision}-graf-{grafana version}-{run-id}
// Example api-tests-ee654f-graf-10.3-load-2024123-140035
func (t *TestRunner) getSuiteRunId(runId string, suite executor.TestSuite) string {
	return fmt.Sprintf("%s-%s-graf-%s-%s",
		suite.Name,
		suite.Revision,
		t.GrafanaVersion,
		runId,
	)
}
