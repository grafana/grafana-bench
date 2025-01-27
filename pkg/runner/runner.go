// Package runner implements the test runners
package runner

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"time"

	"github.com/grafana/grafana-bench/pkg/dashboard"
	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/reporter"
	"github.com/grafana/grafana-bench/pkg/utils/id"
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
	}
}

func (t *TestRunner) Exec(ctx context.Context, testType TestType, suite executor.TestSuite, testVars map[string]string) error {
	var err error

	// get an unique identification for the run
	runId := id.Run(t.Trigger, time.Now())
	t.Log = t.Log.With("runId", runId)

	// get an unique identification for the suite run (used for backward compatibility)
	suiteRunName := id.SuiteRunName(t.Trigger, suite.Name, testType.Name())
	// TODO: remove suiteRun
	t.Log = t.Log.With("suiteRun", suiteRunName)

	// set common test execution variables
	env := map[string]string{
		"TEST_TYPE":              testType.Name(),
		"TEST_SUITE_REVISION":    suite.Revision,
		"GRAFANA_URL":            t.GrafanaInstance.Url(),
		"GRAFANA_ADMIN_USER":     t.GrafanaInstance.AdminUser(),
		"GRAFANA_ADMIN_PASSWORD": t.GrafanaInstance.AdminPassword(),
	}

	maps.Copy(env, testVars)

	suiteRun, err := t.Executor.ExecTestSuite(ctx, suite, env)
	if err != nil {
		return fmt.Errorf("executing test suite: %w", err)
	}

	// TODO: handle error from reporter
	err = t.Reporter.Report(ctx, suite.Name, suite.Revision, runId, suiteRunName, suiteRun)
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

	return nil
}
