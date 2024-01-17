package main

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/bench/provisioner"
)

type TestRunner struct {
	Log              *slog.Logger
	Type             TestType
	Trigger          string
	TestRevision     string
	Tests            []string
	K6CloudToken     string
	K6CloudProjectID string
	GrafanaInstance  *provisioner.VMInstance
	GrafanaRevision  string
	GrafanaTimeout   time.Duration
	GrafanaVersion   string
	MachineSpec      string
}

func NewTestRunner(
	log *slog.Logger,
	testTrigger string,
	testType TestType,
	tests []string,
	testRevision string,
	k6CloudProjectId,
	k6CloudToken string,
	grafanaInstance *provisioner.VMInstance,
	grafanaTimeout time.Duration,
	machineSpec string,
) *TestRunner {

	return &TestRunner{
		Log:              log,
		Trigger:          testTrigger,
		Type:             testType,
		Tests:            tests,
		TestRevision:     testRevision,
		K6CloudToken:     k6CloudToken,
		K6CloudProjectID: k6CloudProjectId,
		GrafanaInstance:  grafanaInstance,
		GrafanaTimeout:   grafanaTimeout,
		MachineSpec:      machineSpec,
	}
}

func (t *TestRunner) Exec(ctx context.Context) error {
	log := t.Log.With("svc", "boot-test-runner")

	// TODO implement a timeout of some sort
	log.Info("Waiting for grafana server...", "address", t.GrafanaInstance.ServiceAddress())

	grafanaCtx, _ := context.WithTimeout(ctx, t.GrafanaTimeout)
	err := t.GrafanaInstance.WaitForLiveGrafana(grafanaCtx)
	if err != nil {
		return fmt.Errorf("checking Grafana is Live... %w", err)
	}
	log.Info("Grafana server is ready!")

	t.GrafanaVersion, err = provisioner.GetGrafanaBuildVersion(t.GrafanaInstance)
	if err != nil {
		return fmt.Errorf("getting grafana version %w", err)
	}

	runIdentifier := t.newRunIdentifier()
	t.Log.Info("suite identifier", "identifier", runIdentifier)

	t.Log = log.With("svc", fmt.Sprintf("%s-test-runner", t.Type.Name()))

	if t.Type == SmokeTest {
		return t.smokeTest(ctx, runIdentifier)
	} else {
		return t.loadTest(ctx, runIdentifier)
	}
}

// newRunIdentifier creates an identifier to be used for
// building dashboards in hosted grafana
//
// smoke-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
// load-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
func (t *TestRunner) newRunIdentifier() string {
	// {type}-{time}-api-tests-{sha}-graf-{version}
	return fmt.Sprintf("%s-%s-api-tests-%s-graf-%s",
		t.Type.Name(),
		time.Now().UTC().Format("15:04:05"),
		t.TestRevision,
		t.GrafanaRevision,
	)
}

// we expect scenarios to be named like the file
// tests/dashboards/dashboard_create.js -> dashboardCreate
func getScenarioName(filename string) string {
	filename = filepath.Base(filename)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.Split(filename, "_")
	for i, p := range parts {
		// don't capitalize the first word
		if i == 0 {
			continue
		}
		parts[i] = strings.Title(p)
	}
	return strings.Join(parts, "")
}

// load test runs 100 iteration(default specified in test suite) of each test
// specified and reports to k6 cloud
func (t *TestRunner) loadTest(ctx context.Context, runIdentifier string) error {
	envVars := map[string]string{
		"MACHINE_SPEC":        t.MachineSpec,
		"TEST_TYPE":           t.Type.Name(),
		"TEST_SUITE_REVISION": t.TestRevision,
		// TODO unify variable names
		"GRAFANA_URL":      t.GrafanaInstance.SchemeServiceAddress(),
		"GRAFANA_USERNAME": t.GrafanaInstance.ServiceUser,
		"GRAFANA_PASSWORD": t.GrafanaInstance.ServicePassword,
		"GT_URL":           t.GrafanaInstance.SchemeServiceAddress(),
		"GT_USERNAME":      t.GrafanaInstance.ServiceUser,
		"GT_PASSWORD":      t.GrafanaInstance.ServicePassword,
		//

		"K6_CLOUD_TOKEN":          t.K6CloudToken,
		"K6_CLOUD_PROJECT_ID":     t.K6CloudProjectID,
		"K6_CLOUD_TRACES_ENABLED": "true",
	}

	// run k6 tests
	var (
		startTime     = time.Now()
		totalDuration float32
		anyFailures   = false
	)

	// run the tests
	for iteration, testFile := range t.Tests {
		scenarioName := getScenarioName(testFile)
		// set the scenario name so it's accessible from the test
		envVars["SCENARIO_NAME"] = scenarioName

		// run command send output to cloud
		k6Summary, err := K6ExecTest(
			t.Log,
			testFile,
			scenarioName,
			runIdentifier,
			envVars,
			"--out", "cloud",
		)
		if err != nil {
			t.Log.Error("executing k6 test", "error", err)
			// TODO: maybe we should break the iteration here, as test result may not be relevant
		}

		totalDuration += k6Summary.Durations.TotalDuration

		// test complete log
		t.Log.Info("testRun",
			// TODO: remove BenchRevision is no longer relevant
			// "benchVersion", ps.BenchRevision,
			"apiTestsVersion", t.TestRevision,
			"testRun", t.newTestIdentifier(testFile),
			"suiteRun", runIdentifier,
			"scenarioName", scenarioName,
			"grafanaUrl", t.GrafanaInstance.Host,
			"grafanaVersion", t.GrafanaVersion,
			"folder", path.Dir(testFile),
			"testFile", path.Base(testFile),
			"order", strconv.Itoa(iteration+1),
			"iterations", k6Summary.Iterations,
			"k6CloudUrl", k6Summary.CloudURL,
			"k6CloudID", k6Summary.CloudId,
			"setupDuration", prettyMS(k6Summary.Durations.SetupDuration),
			"scenarioDuration", prettyMS(k6Summary.Durations.ScenarioDuration),
			"teardownDuration", prettyMS(k6Summary.Durations.TeardownDuration),
			"totalDuration", prettyMS(k6Summary.Durations.TotalDuration),
			"exitMessage", k6Summary.ExitMessage,
			"exitCode", k6Summary.ExitCode,
		)
	}

	totalScenarioDurations := prettyMS(totalDuration)
	benchDuration := prettyMS(float32(time.Since(startTime).Milliseconds()))

	t.Log.Info("suiteRun",
		// TODO: remove BenchRevision is no longer relevant
		// "benchVersion", ps.BenchRevision,
		"apiTestsVersion", t.TestRevision,
		"suiteRun", runIdentifier,
		"testTrigger", t.Trigger,
		"grafanaUrl", t.GrafanaInstance.Host,
		"grafanaVersion", t.GrafanaRevision,
		"startTime", startTime.Format(time.RFC3339),
		"totalScenarioDurations", totalScenarioDurations,
		"duration", benchDuration,
		"machineInfo", t.MachineSpec,
		"anyFailures", anyFailures,
	)

	return nil
}

// smokeTest runs a single iteration of each test specified and does not report
// to k6 cloud
func (t *TestRunner) smokeTest(ctx context.Context, runIdentifier string) error {
	envVars := map[string]string{
		"MACHINE_SPEC":        t.MachineSpec,
		"TEST_TYPE":           t.Type.Name(),
		"TEST_SUITE_REVISION": t.TestRevision,
		"GT_URL":              t.GrafanaInstance.SchemeServiceAddress(),
		"GT_USERNAME":         t.GrafanaInstance.ServiceUser,
		"GT_PASSWORD":         t.GrafanaInstance.ServicePassword,
	}

	var (
		startTime     = time.Now()
		totalDuration float32
		anyFailures   = false
	)

	// run the tests
	for iteration, testFile := range t.Tests {
		scenarioName := getScenarioName(testFile)

		// set the scenario name so it's accessible from the test
		envVars["SCENARIO_NAME"] = scenarioName

		// run command
		k6Summary, err := K6ExecTest(
			t.Log,
			testFile,
			scenarioName,
			runIdentifier,
			envVars,
		)
		if err != nil {
			t.Log.Error("executing k6 test", "error", err)
			// TODO: maybe we should break the iteration here, as test result may not be relevant
		}

		totalDuration += k6Summary.Durations.TotalDuration

		// test complete log
		t.Log.Info("testRun",
			// TODO: remove BenchRevision is no longer relevant
			// "benchVersion", ps.BenchRevision,
			"apiTestsVersion", t.TestRevision,
			"testRun", t.newTestIdentifier(testFile),
			"suiteRun", runIdentifier,
			"scenarioName", scenarioName,
			"grafanaUrl", t.GrafanaInstance.Host,
			"grafanaVersion", t.GrafanaRevision,
			"folder", path.Dir(testFile),
			"testFile", path.Base(testFile),
			"order", strconv.Itoa(iteration+1),
			"iterations", k6Summary.Iterations,
			"setupDuration", prettyMS(k6Summary.Durations.SetupDuration),
			"scenarioDuration", prettyMS(k6Summary.Durations.ScenarioDuration),
			"teardownDuration", prettyMS(k6Summary.Durations.TeardownDuration),
			"totalDuration", prettyMS(k6Summary.Durations.TotalDuration),
			"exitMessage", k6Summary.ExitMessage,
			"exitCode", k6Summary.ExitCode,
		)
	}

	totalScenarioDurations := prettyMS(totalDuration)
	benchDuration := prettyMS(float32(time.Since(startTime).Milliseconds()))

	t.Log.Info("suiteRun",
		// FIXME: BenchRevision is no longer relevant
		//	"benchVersion", ps.BenchRevision,
		"apiTestsVersion", t.TestRevision,
		"suiteRun", runIdentifier,
		"testTrigger", t.Trigger,
		"grafanaUrl", t.GrafanaInstance.Host,
		"grafanaVersion", t.GrafanaRevision,
		"startTime", startTime.Format(time.RFC3339),
		"totalScenarioDurations", totalScenarioDurations,
		"duration", benchDuration,
		"machineInfo", t.MachineSpec,
		"anyFailures", anyFailures,
	)

	if anyFailures {
		// TODO: remove reference to dashboard. This seems particular to the hosted grafana R
		dashboardUrl := fmt.Sprintf("https://ops.grafana-ops.net/d/d3381df1-fa32-4955-994a-e6a8bca58025/test-runs?var-SuiteRun=%s", runIdentifier)
		return fmt.Errorf("Smoke test failed. Too many test failures. Review logs or see dashboard %s", dashboardUrl)
	}

	return nil
}

// dashboardCreate.js-13:37:35-smoke-api-tests-cb5adc0-graf-10.2.0-60657
func (t *TestRunner) newTestIdentifier(filename string) string {
	// {filename}-{time}-{type}-api-tests-{sha}-graf-{version}
	return fmt.Sprintf("%s-%s-%s-api-tests-%s-graf-%s",
		filepath.Base(filename),
		time.Now().UTC().Format("15:04:05"),
		t.Type.Name(),
		t.TestRevision,
		t.GrafanaVersion,
	)
}
