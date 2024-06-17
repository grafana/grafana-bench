package test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"text/template"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/grafana"
	"github.com/grafana/grafana-bench/pkg/reporter"
)

type TestRunner struct {
	Log             *slog.Logger
	Trigger         string
	GrafanaInstance grafana.GrafanaInstance
	GrafanaVersion  string
	MachineSpec     string
	BenchRevision   string
	DashboardURL    string
	Executor        executor.TestExecutor
	ReportFormat    string
}

func 	NewTestRunner(
	log *slog.Logger,
	testTrigger string,
	grafanaInstance grafana.GrafanaInstance,
	machineSpec string,
	benchRevision string,
	dashboardURL string,
	executor executor.TestExecutor,
	reportFormat string,

) *TestRunner {
	return &TestRunner{
		Log:             log.With("svc", "test-runner"),
		Trigger:         testTrigger,
		GrafanaInstance: grafanaInstance,
		MachineSpec:     machineSpec,
		BenchRevision:   benchRevision,
		DashboardURL:    dashboardURL,
		Executor:        executor,
		ReportFormat:    reportFormat,
	}
}

func (t *TestRunner) Exec(ctx context.Context, testType TestType, suite executor.TestSuite, testVars map[string]string) error {
	// get an unique identification for the run
	runId := t.getRunId(testType)
	t.Log = t.Log.With("runId", runId)

	t.Log.Info("Waiting for grafana server...", "address", t.GrafanaInstance.Url())

	err := t.GrafanaInstance.WaitForLiveGrafana(ctx)
	if err != nil {
		return fmt.Errorf("checking Grafana is Live... %w", err)
	}
	t.Log.Debug("Grafana server is ready!")

	t.GrafanaVersion, err = t.GrafanaInstance.GetGrafanaBuildVersion()
	if err != nil {
		return fmt.Errorf("getting grafana version: %w", err)
	}

	// get an unique identification for the suite run (used for backward compatibility)
	suiteRunId := t.getSuiteRunId(runId, suite)
	t.Log = t.Log.With("suiteRun", suiteRunId)

	suiteReporter, err := t.getReporter()
	if err != nil {
		return fmt.Errorf("getting reporter %w", err)
	}

	// set common test execution variables
	env := map[string]string{
		"MACHINE_SPEC":        t.MachineSpec,
		"TEST_TYPE":           testType.Name(),
		"TEST_SUITE_REVISION": suite.Revision,
		// TODO unify variable names
		"GRAFANA_URL":      t.GrafanaInstance.Url(),
		"GRAFANA_USERNAME": t.GrafanaInstance.UserName(),
		"GRAFANA_PASSWORD": t.GrafanaInstance.Password(),
		"GT_URL":           t.GrafanaInstance.Url(),
		"GT_USERNAME":      t.GrafanaInstance.UserName(),
		"GT_PASSWORD":      t.GrafanaInstance.Password(),
		// ----
	}

	maps.Copy(env, testVars)

	suiteRun, err := t.Executor.ExecTestSuite(ctx, suite, env)
	if err != nil {
		return fmt.Errorf("executing test suite %w", err)
	}

	suiteReporter.Report(runId,suiteRunId, suite, suiteRun)

	var anyFailures = suiteRun.Status != executor.SuitePassed

	if anyFailures {
		dashboardMsg := ""
		if t.DashboardURL != "" {
			// NOTE this block of code performs substitution on a user defined url. e.g.
			// http://mygrafana.com/b/?suiteRun={suiteRun}
			// This functionality is ALPHA and may be removed in favor of outputting
			// the suiteRun ID and leaving it up to the user.
			dashboard, err := t.getDashboardURL(runId)
			if err != nil {
				t.Log.With(t.testRunnerLogAttrs()...).
					Error("getting URL dashboard: %w", err)
			} else {
				dashboardMsg = fmt.Sprintf(". See dashboard: %s", dashboard)
			}
		}

		return fmt.Errorf("test suite failed: Too many test failures%s", dashboardMsg)
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

// testRunnerLogAttrs formats the test runner attributes as log attributes
// TODO: check for missing attributes (for example add test type?)
func (t *TestRunner) testRunnerLogAttrs() []any {
	return []any{
		"testTrigger", t.Trigger,
		"benchRevision", t.BenchRevision,
		//TODO: deprecate this attribute
		"grafanaUrl", t.GrafanaInstance.Hostname(),
		"grafanSlug", t.GrafanaInstance.Slug(),
		"grafanaVersion", t.GrafanaVersion,
		"testExecutor", t.Executor.Name(),
	}
}


// getDashboardURL takes t.DashboardURL and substitutes {{.SuiteRun}} for t.RunIdentifier
// this functionality may be deprecated in the future.
func (t *TestRunner) getDashboardURL(runIdentifier string) (string, error) {
	if t.DashboardURL == "" {
		return "", fmt.Errorf("URL template is empty")
	}

	template, err := template.New("dashboard").Parse(t.DashboardURL)
	if err != nil {
		return "", fmt.Errorf("error parsing template %w", err)
	}

	// substitution variables
	// TODO: define more substitution variables
	vars := struct {
		SuiteRun string
	}{
		SuiteRun: runIdentifier,
	}

	dashboardURL := bytes.Buffer{}
	err = template.Execute(&dashboardURL, vars)
	if err != nil {
		return "", fmt.Errorf("invalid template substitution: %w", err)
	}

	return dashboardURL.String(), nil
}

func (t *TestRunner) getReporter() (reporter.SuiteRunReporter, error) {
	switch t.ReportFormat {
	case "log": return reporter.NewLogReporter(t.testRunnerLogAttrs()), nil
	case "text": return reporter.NewTextReporter(os.Stdout), nil
	default: return nil, fmt.Errorf("invalid report format %q", t.ReportFormat)
	}
}