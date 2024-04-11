package test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"text/template"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/grafana"
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
}

func NewTestRunner(
	log *slog.Logger,
	testTrigger string,
	grafanaInstance grafana.GrafanaInstance,
	machineSpec string,
	benchRevision string,
	dashboardURL string,
	executor executor.TestExecutor,

) *TestRunner {
	return &TestRunner{
		Log:             log,
		Trigger:         testTrigger,
		GrafanaInstance: grafanaInstance,
		MachineSpec:     machineSpec,
		BenchRevision:   benchRevision,
		DashboardURL:    dashboardURL,
		Executor:        executor,
	}
}

func (t *TestRunner) Exec(ctx context.Context, testType TestType, suite executor.TestSuite) error {
	// get an unique identification for the run
	runId := t.getRunId(testType)
	t.Log = t.Log.With("runId", runId)

	t.Log.With(suiteLogAttrs(suite)...).Info("starting suite run")

	t.Log.Info("Waiting for grafana server...", "address", t.GrafanaInstance.Address())

	err := t.GrafanaInstance.WaitForLiveGrafana(ctx)
	if err != nil {
		return fmt.Errorf("checking Grafana is Live... %w", err)
	}
	t.Log.Info("Grafana server is ready!")

	t.GrafanaVersion, err = t.GrafanaInstance.GetGrafanaBuildVersion()
	if err != nil {
		return fmt.Errorf("getting grafana version: %w", err)
	}

	// get an unique identification for the suite run (used for backward compatibility)
	suiteRunId := t.getSuiteRunId(runId, suite)
	t.Log = t.Log.With("suiteRun", suiteRunId)

	t.Log.With(suiteLogAttrs(suite)...).Info("starting suite run")

	// set common test execution variables
	env := map[string]string{
		"MACHINE_SPEC":        t.MachineSpec,
		"TEST_TYPE":           testType.Name(),
		"TEST_SUITE_REVISION": suite.Revision,
		// TODO unify variable names
		"GRAFANA_URL":      t.GrafanaInstance.Address(),
		"GRAFANA_USERNAME": t.GrafanaInstance.UserName(),
		"GRAFANA_PASSWORD": t.GrafanaInstance.Password(),
		"GT_URL":           t.GrafanaInstance.Address(),
		"GT_USERNAME":      t.GrafanaInstance.UserName(),
		"GT_PASSWORD":      t.GrafanaInstance.Password(),
		// ----
	}

	suiteRun, err := t.Executor.ExecTestSuite(ctx, suite, env)
	if err != nil {
		return fmt.Errorf("executing test suite %w", err)
	}

	for _, testRun := range suiteRun.TestRuns {
		testRunId := fmt.Sprintf("%s-%d", runId, testRun.Order)
		t.Log.With(t.testRunnerLogAttrs()...).
			With(suiteLogAttrs(suite)...).
			With(testRunLogAttrs(testRun)...).
			Info("testRun", "testRun", testRunId)
	}

	var anyFailures = (suiteRun.TestsFailed + suiteRun.TestsError) > 0

	t.Log.With(t.testRunnerLogAttrs()...).
		With(suiteLogAttrs(suite)...).
		With(suiteRunLogAttrs(suiteRun)...).
		Info("suiteRun", "anyFailures", anyFailures)

	// NOTE this block of code performs substitution on a user defined url. e.g.
	// http://mygrafana.com/b/?suiteRun={suiteRun}
	// This functionality is ALPHA and may be removed in favor of outputting
	// the suiteRun ID and leaving it up to the user.
	if anyFailures {
		var dashboardMsg string
		if t.DashboardURL != "" {
			dashboard, err := t.getDashboardURL(runId)
			if err != nil {
				return fmt.Errorf("getting URL dashboard: %w", err)
			}
			dashboardMsg = " See dashboard: " + dashboard
		}

		t.Log.With(suiteLogAttrs(suite)...).
			Error("test suite failed. Too many test failures." + dashboardMsg)
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
		"grafanaUrl", t.GrafanaInstance.Address(),
		"grafanaVersion", t.GrafanaVersion,
		"testExecutor", t.Executor.Name(),
	}
}

// suiteLogAttrs formats suite's attributes as log attributes
func suiteLogAttrs(suite executor.TestSuite) []any {
	return []any{
		"suiteId", fmt.Sprintf("%s-%s", suite.Name, suite.Revision),
		"suiteIdName", suite.Name,
		"suiteRevision", suite.Revision,
	}
}

// suiteRunLogAttrs formats suite run's attributes as log attributes
func suiteRunLogAttrs(suiteRun executor.SuiteRunSummary) []any {
	return []any{
		"startTime", suiteRun.StartTime.Format(time.RFC3339),
		"totalScenarioDurations", suiteRun.ScenariosDuration,
		"duration", suiteRun.TotalDuration,
		"testsExecuted", suiteRun.TestsExecuted,
		"testsPassed", suiteRun.TestsPassed,
		"testsFailed", suiteRun.TestsFailed,
		"testsError", suiteRun.TestsError,
	}
}

// testRunLogAttrs returns the k6RunSummary attributes formatted as log attributes
func testRunLogAttrs(testRun executor.TestRun) []any {
	attrs := []any{
		"folder", testRun.TestFolder,
		"testFile", testRun.TestFile,
		"order", strconv.Itoa(testRun.Order),
		"iterations", testRun.Iterations,
		"setupDuration", prettyMS(testRun.Durations.SetupDuration),
		"scenarioDuration", prettyMS(testRun.Durations.ScenarioDuration),
		"teardownDuration", prettyMS(testRun.Durations.TeardownDuration),
		"totalDuration", prettyMS(testRun.Durations.TotalDuration),
		"status", testRun.Status,
		"exitMessage", testRun.ExitMessage,
		"exitCode", strconv.Itoa(testRun.ExitCode),
	}

	for k, v := range testRun.Attributes {
		attrs = append(attrs, k, v)
	}
	return attrs
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

// prettyMS adds ms suffix to ms float
func prettyMS(ms float32) string {
	duration := time.Duration(ms) * time.Millisecond
	return fmt.Sprintf("%dms", duration.Milliseconds())
}
