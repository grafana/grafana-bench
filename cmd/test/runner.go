package test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils"
)

type TestRunner struct {
	Log               *slog.Logger
	Verbose           bool
	K6CloudOutput     bool
	RunIdentifier     string
	Type              TestType
	Trigger           string
	TestSuiteBase     string
	TestSuite         string
	TestSuiteName     string
	TestSuiteRevision string
	K6CloudToken      string
	K6CloudProjectID  string
	GrafanaInstance   *provisioner.VMInstance
	GrafanaTimeout    time.Duration
	GrafanaVersion    string
	MachineSpec       string
	BenchRevision     string
	DashboardURL      string
}

func NewTestRunner(
	log *slog.Logger,
	verbose bool,
	cloudOutput bool,
	testTrigger string,
	testType TestType,
	testSuiteBase string,
	testSuite string,
	testSuiteName,
	testRevision string,
	k6CloudProjectId,
	k6CloudToken string,
	grafanaInstance *provisioner.VMInstance,
	grafanaTimeout time.Duration,
	machineSpec string,
	benchRevision string,
	dashboardURL string,
) *TestRunner {
	return &TestRunner{
		Log:               log,
		Verbose:           verbose,
		K6CloudOutput:     cloudOutput,
		Trigger:           testTrigger,
		Type:              testType,
		TestSuiteBase:     testSuiteBase,
		TestSuite:         testSuite,
		TestSuiteName:     testSuiteName,
		TestSuiteRevision: testRevision,
		K6CloudToken:      k6CloudToken,
		K6CloudProjectID:  k6CloudProjectId,
		GrafanaInstance:   grafanaInstance,
		GrafanaTimeout:    grafanaTimeout,
		MachineSpec:       machineSpec,
		BenchRevision:     benchRevision,
		DashboardURL:      dashboardURL,
	}
}

func (t *TestRunner) Exec(ctx context.Context) error {
	t.Log.Info("Waiting for grafana server...", "address", t.GrafanaInstance.ServiceAddress())

	grafanaCtx, cancel := context.WithTimeout(ctx, t.GrafanaTimeout)
	defer cancel()

	err := t.GrafanaInstance.WaitForLiveGrafana(grafanaCtx)
	if err != nil {
		return fmt.Errorf("checking Grafana is Live... %w", err)
	}
	t.Log.Info("Grafana server is ready!")

	t.GrafanaVersion, err = provisioner.GetGrafanaBuildVersion(t.GrafanaInstance)
	if err != nil {
		return fmt.Errorf("getting grafana version %w", err)
	}

	t.RunIdentifier = t.newRunIdentifier()
	t.Log.Info("suite identifier", "identifier", t.RunIdentifier)

	k6Version, err := GetK6Version()
	if err != nil {
		return fmt.Errorf("getting k6 version %w", err)
	}

	t.Log.Info("using k6", "k6Version", k6Version)

	if t.Type == SmokeTest {
		return t.smokeTest(ctx)
	} else {
		return t.loadTest(ctx)
	}
}

// newRunIdentifier creates an identifier to link tests together when querying
// test output using the format {type}-{time}-{test suite}-{sha}-graf-{version}
// Examples:
//
//	smoke-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
//	load-13:37:35-api-tests-cb5adc0-graf-10.2.0-60657
func (t *TestRunner) newRunIdentifier() string {
	return fmt.Sprintf("%s-%s-%s-%s-graf-%s",
		t.Type.Name(),
		time.Now().UTC().Format("15:04:05"),
		t.TestSuiteName,
		t.TestSuiteRevision,
		t.GrafanaVersion,
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

// suiteRunLogAttrs formats the test runner's attributes as log attributes
// TODO: check for missing attributes (for example add test type?)
func (t *TestRunner) suiteRunLogAttrs() []any {
	return []any{
		"testTrigger", t.Trigger,
		"benchRevision", t.BenchRevision,
		"testSuiteRevision", t.TestSuiteRevision,
		"suiteRun", t.RunIdentifier,
		"grafanaUrl", t.GrafanaInstance.Host,
		"grafanaVersion", t.GrafanaVersion,
	}
}

// testRunLogAttrs returns the k6RunSummary attributes formatted as log attributes
func testRunLogAttrs(k K6RunSummary) []any {
	return []any{
		"iterations", k.Iterations,
		"k6CloudUrl", k.CloudURL,
		"k6CloudID", k.CloudId,
		"setupDuration", prettyMS(k.Durations.SetupDuration),
		"scenarioDuration", prettyMS(k.Durations.ScenarioDuration),
		"teardownDuration", prettyMS(k.Durations.TeardownDuration),
		"totalDuration", prettyMS(k.Durations.TotalDuration),
		"exitMessage", k.ExitMessage,
		"exitCode", strconv.Itoa(k.ExitCode),
	}
}

// TestSuiteRunSummary summary
type TestSuiteRunSummary struct {
	AnyFailures   bool
	TotalDuration float32
}

// load test runs 100 iteration(default specified in test suite) of each test
// specified and reports to k6 cloud if credentials are set
func (t *TestRunner) loadTest(ctx context.Context) error {
	// k6 cloud environment variables
	var (
		k6Env  map[string]string
		k6Args []string
	)
	if t.K6CloudOutput {
		if t.K6CloudProjectID == "" || t.K6CloudToken == "" {
			return fmt.Errorf("k6 Token and project ID are required for cloud output")
		}

		k6Env = map[string]string{
			"K6_CLOUD_TOKEN":          t.K6CloudToken,
			"K6_CLOUD_PROJECT_ID":     t.K6CloudProjectID,
			"K6_CLOUD_TRACES_ENABLED": "true",
		}
		k6Args = []string{"--out", "cloud"}
	} else {
		t.Log.Warn("running load tests with cloud output disabled.")
	}

	// exec test and redirect output to cloud
	_, err := t.execTest(ctx, k6Env, k6Args...)

	return err
}

// smokeTest runs a single iteration of each test specified and does not report
// to k6 cloud
func (t *TestRunner) smokeTest(ctx context.Context) error {
	summary, err := t.execTest(ctx, map[string]string{})
	if err != nil {
		return err
	}

	// NOTE this block of code performs substitution on a user defined url. e.g.
	// http://mygrafana.com/b/?suiteRun={suiteRun}
	// This functionality is ALPHA and may be removed in favor of outputting
	// the suiteRun ID and leaving it up to the user.
	if summary.AnyFailures {
		var dashboardMsg string
		if t.DashboardURL != "" {
			dashboard, err := t.getDashboardURL()
			if err != nil {
				return fmt.Errorf("getting URL dashboard: %w", err)
			}
			dashboardMsg = " See dashboard: " + dashboard
		}

		t.Log.With(t.suiteRunLogAttrs()...).Error("test suite failed. Too many test failures." + dashboardMsg)
	}

	return nil
}

// getDashboardURL takes t.DashboardURL and substitutes {{.SuiteRun}} for t.RunIdentifier
// this functionality may be deprecated in the future.
func (t *TestRunner) getDashboardURL() (string, error) {
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
		SuiteRun: t.RunIdentifier,
	}

	dashboardURL := bytes.Buffer{}
	err = template.Execute(&dashboardURL, vars)
	if err != nil {
		return "", fmt.Errorf("invalid template substitution: %w", err)
	}

	return dashboardURL.String(), nil
}

// execute test suite
func (t *TestRunner) execTest(ctx context.Context, env map[string]string, args ...string) (TestSuiteRunSummary, error) {
	// set common test execution variables
	envVars := map[string]string{
		"MACHINE_SPEC":        t.MachineSpec,
		"TEST_TYPE":           t.Type.Name(),
		"TEST_SUITE_REVISION": t.TestSuiteRevision,
		// TODO unify variable names
		"GRAFANA_URL":      t.GrafanaInstance.SchemeServiceAddress(),
		"GRAFANA_USERNAME": t.GrafanaInstance.ServiceUser,
		"GRAFANA_PASSWORD": t.GrafanaInstance.ServicePassword,
		"GT_URL":           t.GrafanaInstance.SchemeServiceAddress(),
		"GT_USERNAME":      t.GrafanaInstance.ServiceUser,
		"GT_PASSWORD":      t.GrafanaInstance.ServicePassword,
		//
	}

	// add or override environment variables passed to execution
	for k, v := range env {
		envVars[k] = v
	}

	// run k6 tests
	var (
		startTime     = time.Now()
		totalDuration float32
		anyFailures   = false
	)

	tests, err := t.getTestFiles()
	if err != nil {
		return TestSuiteRunSummary{}, fmt.Errorf("getting test list: %w", err)
	}
	// run the tests
	for iteration, testFile := range tests {
		scenarioName := getScenarioName(testFile)
		// set the scenario name so it's accessible from the test
		envVars["SCENARIO_NAME"] = scenarioName

		// run command send output to cloud
		k6Summary, err := K6ExecTest(
			t.Log,
			t.Verbose,
			t.K6CloudOutput,
			testFile,
			scenarioName,
			t.RunIdentifier,
			envVars,
			args...,
		)
		if err != nil {
			t.Log.Error("executing k6 test", "error", err)
			// TODO: maybe we should break the iteration here, as test result may not be relevant
		}

		totalDuration += k6Summary.Durations.TotalDuration
		anyFailures = anyFailures || k6Summary.AnyFailures

		// get the path to the test relative to the TestSuiteBase if any
		// we don't need to check for errors because how the test path is constructed
		rootDir, _ := filepath.Abs(t.TestSuiteBase)
		testFolder, _ := filepath.Rel(rootDir, filepath.Dir(testFile))

		// test complete log
		testTags := []any{
			"testRun", t.newTestIdentifier(testFile),
			"scenarioName", scenarioName,
			"folder", testFolder,
			"testFile", path.Base(testFile),
			"order", strconv.Itoa(iteration + 1),
		}
		t.Log.With(t.suiteRunLogAttrs()...).
			With(testTags...).
			Info("testrun", testRunLogAttrs(k6Summary)...)
	}

	totalScenarioDurations := prettyMS(totalDuration)
	benchDuration := prettyMS(float32(time.Since(startTime).Milliseconds()))

	t.Log.With(t.suiteRunLogAttrs()...).Info("suiteRun",
		"startTime", startTime.Format(time.RFC3339),
		"totalScenarioDurations", totalScenarioDurations,
		"duration", benchDuration,
		"anyFailures", anyFailures,
	)

	return TestSuiteRunSummary{
		AnyFailures:   anyFailures,
		TotalDuration: totalDuration,
	}, nil
}

// returns a test identifier of the form {filename}-{time}-{type}-{test suite}-{sha}-graf-{version}
// Example: dashboardCreate.js-13:37:35-smoke-api-tests-cb5adc0-graf-10.2.0-60657
func (t *TestRunner) newTestIdentifier(filename string) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s-graf-%s",
		filepath.Base(filename),
		time.Now().UTC().Format("15:04:05"),
		t.Type.Name(),
		t.TestSuiteName,
		t.TestSuiteRevision,
		t.GrafanaVersion,
	)
}

// getTestFiles returns the list of tests to execute.
// If tests points to a file with a js extension run that single file.
// If it points to a directory all of the .js files in it are recursively searched.
// tests=dashboard_read.js will run dashboard_read.js.
// tests=dashboards will run all files in dashboards/**.*.js.
func (t *TestRunner) getTestFiles() ([]string, error) {
	if filepath.IsAbs(t.TestSuite) {
		return nil, fmt.Errorf("test suite must be a relative to base dir. Got %q", t.TestSuite)
	}

	testSuitePath, err := filepath.Abs(path.Join(t.TestSuiteBase, t.TestSuite))
	if err != nil {
		return nil, fmt.Errorf("getting path to test suite %w", err)
	}

	exists, _ := utils.PathExists(testSuitePath)
	if !exists {
		return nil, fmt.Errorf("test suite %s not found", testSuitePath)
	}

	fileInfo, err := os.Stat(testSuitePath)
	if err != nil {
		return nil, fmt.Errorf("opening test suite at %s: %w", testSuitePath, err)
	}

	// test suite points to a directory
	if fileInfo.IsDir() {
		files, err := utils.GlobByExtension(testSuitePath, ".js")
		if err != nil {
			return nil, err
		}

		if len(files) == 0 {
			return nil, fmt.Errorf("no test files found at %s", testSuitePath)
		}

		return files, nil
	}

	// is a file, we expect a single .js file
	testSuiteFile := path.Base(testSuitePath)
	if !strings.HasSuffix(testSuiteFile, ".js") {
		return nil, fmt.Errorf("expected a .js file got %s", testSuiteFile)
	}

	return []string{testSuitePath}, nil
}
