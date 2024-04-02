package test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"text/template"
	"time"

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
	Executor        TestExecutor
}

func NewTestRunner(
	log *slog.Logger,
	testTrigger string,
	grafanaInstance grafana.GrafanaInstance,
	machineSpec string,
	benchRevision string,
	dashboardURL string,
	executor TestExecutor,
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

func (t *TestRunner) Exec(ctx context.Context, testType TestType, suite TestSuite) error {
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

	runIdentifier := t.newRunIdentifier(testType, suite)
	t.Log.Info("suite identifier", "identifier", runIdentifier)

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
		t.Log.With(t.suiteLogAttrs(suite, runIdentifier)...).
			With(t.testRunLogAttrs(testRun)...).
			Info("testRun", "testRun", t.newTestIdentifier(testType, suite, testRun))
	}

	var anyFailures = (suiteRun.TestsFailed + suiteRun.TestsError) > 0

	t.Log.With(t.suiteLogAttrs(suite, runIdentifier)...).
		With(t.suiteRunLogAttrs(suiteRun)...).
		Info("suiteRun", "anyFailures", anyFailures)

	// NOTE this block of code performs substitution on a user defined url. e.g.
	// http://mygrafana.com/b/?suiteRun={suiteRun}
	// This functionality is ALPHA and may be removed in favor of outputting
	// the suiteRun ID and leaving it up to the user.
	if anyFailures {
		var dashboardMsg string
		if t.DashboardURL != "" {
			dashboard, err := t.getDashboardURL(runIdentifier)
			if err != nil {
				return fmt.Errorf("getting URL dashboard: %w", err)
			}
			dashboardMsg = " See dashboard: " + dashboard
		}

		t.Log.With(t.suiteLogAttrs(suite, runIdentifier)...).
			Error("test suite failed. Too many test failures." + dashboardMsg)
	}

	return nil
}

// newRunIdentifier creates an identifier to link tests together when querying
// test output using the format {type}-{time}-{test suite}-{sha}-graf-{version}
// Examples:
//
//	smoke-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
//	load-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
func (t *TestRunner) newRunIdentifier(testType TestType, suite TestSuite) string {
	return fmt.Sprintf("%s-%s-%s-%s-graf-%s",
		testType.Name(),
		time.Now().UTC().Format("15:04:05"),
		suite.Name,
		suite.Revision,
		t.GrafanaVersion,
	)
}

// TestIdentifier returns a test identifier of the form {filename}-{time}-{type}-{test suite}-{sha}-graf-{version}
// Example: dashboardCreate.js-13:37:35-smoke-api-tests-cb5adc0-graf-10.2.0-60657
func (t *TestRunner) newTestIdentifier(testType TestType, suite TestSuite, testRun TestRun) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s-graf-%s",
		testRun.TestFile,
		testRun.StartTime.UTC().Format("15:04:05"),
		testType.Name(),
		suite.Name,
		suite.Revision,
		t.GrafanaVersion,
	)
}

// suiteRunLogAttrs formats the test suite's attributes as log attributes
// TODO: check for missing attributes (for example add test type?)
func (t *TestRunner) suiteLogAttrs(suite TestSuite, runIdentifier string) []any {
	return []any{
		"suiteRun", runIdentifier,
		"testTrigger", t.Trigger,
		"benchRevision", t.BenchRevision,
		"testSuiteRevision", suite.Revision,
		"grafanaUrl", t.GrafanaInstance.Address(),
		"grafanaVersion", t.GrafanaVersion,
		"testExecutor", t.Executor.Name(),
	}
}

// suiteRunLogAttrs formats the test runner's attributes as log attributes
// TODO: check for missing attributes (for example add test type?)
func (t *TestRunner) suiteRunLogAttrs(suiteRun SuiteRunSummary) []any {
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
func (t *TestRunner) testRunLogAttrs(testRun TestRun) []any {
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
