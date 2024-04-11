package test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	e "github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils"
)

const (
	ThresholdFailed = 99 // return code when test thresholds fail
)

var (
	missingK6CloudConfigError = errors.New("k6 Token and project ID are required for cloud output")
	testFilesError            = errors.New("getting test files")
)

// K6TestExecutor implements TestExecutor interface for running k6 test suites
type K6TestExecutor struct {
	Log            *slog.Logger
	Verbose        bool
	CloudOutput    bool
	CloudToken     string
	CloudProjectID string
}

// NewK6TestExecutor creates a new instance of K6TestExecutor
func NewK6TestExecutor(
	log *slog.Logger,
	verbose bool,
	cloudOutput bool,
	cloudToken string,
	cloudProjectID string,
) *K6TestExecutor {
	return &K6TestExecutor{
		Log:            log.With("executor", "k6"),
		Verbose:        verbose,
		CloudOutput:    cloudOutput,
		CloudToken:     cloudToken,
		CloudProjectID: cloudProjectID,
	}
}

// K6TestRun summarizes the execution of a k6 test
type K6TestRun struct {
	Status      e.TestStatus
	ExitCode    int
	ExitMessage string
	Iterations  string
	Durations   e.TestDurations
	CloudID     string
	CloudURL    string
}

func (t *K6TestExecutor) Name() string {
	return "k6"
}

// execute test suite
func (t *K6TestExecutor) ExecTestSuite(
	ctx context.Context,
	suite e.TestSuite,
	env map[string]string,
) (e.SuiteRunSummary, error) {
	if t.CloudOutput && (t.CloudToken == "" || t.CloudProjectID == "") {
		return e.SuiteRunSummary{}, missingK6CloudConfigError
	}

	// set common test execution variables
	k6env := map[string]string{}

	// copy environment variables passed to execution
	for k, v := range env {
		k6env[k] = v
	}

	// run k6 tests
	var (
		suiteStartTime    = time.Now()
		scenariosDuration float32
	)

	k6Version, err := t.getK6Version()
	if err != nil {
		return e.SuiteRunSummary{}, fmt.Errorf("getting k6 version %w", err)
	}

	t.Log.Info("using k6", "k6Version", k6Version)

	tests, err := t.getTestFiles(suite)
	if err != nil {
		return e.SuiteRunSummary{}, fmt.Errorf("%w: %w", testFilesError, err)
	}

	suiteSummary := e.SuiteRunSummary{}

	// run the tests
	for order, testFile := range tests {
		testStartTime := time.Now()

		scenarioName := getScenarioName(testFile)
		// set the scenario name so it's accessible from the test
		k6env["SCENARIO_NAME"] = scenarioName

		// run command send output to cloud
		k6Summary, err := t.execTest(
			ctx,
			testFile,
			scenarioName,
			k6env,
		)
		if err != nil {
			t.Log.Error("executing k6 test %w", err)
			// TODO: maybe we should break the iteration here, as test result may not be relevant
		}

		scenariosDuration += k6Summary.Durations.TotalDuration

		// get the path to the test relative to the TestSuiteBase if any
		// we don't need to check for errors because how the test path is constructed
		rootDir, _ := filepath.Abs(suite.BaseDir)
		testFolder, _ := filepath.Rel(rootDir, filepath.Dir(testFile))

		summary := e.TestRun{
			TestFolder:  testFolder,
			TestFile:    path.Base(testFile),
			StartTime:   testStartTime,
			Order:       order + 1,
			Status:      k6Summary.Status,
			ExitCode:    k6Summary.ExitCode,
			Durations:   k6Summary.Durations,
			Iterations:  k6Summary.Iterations,
			ExitMessage: k6Summary.ExitMessage,
			Attributes: map[string]string{
				"cloudId":  k6Summary.CloudID,
				"cloudURL": k6Summary.CloudURL,
			},
		}

		suiteSummary.TestsExecuted += 1
		switch summary.Status {
		case e.TestPassed:
			suiteSummary.TestsPassed += 1
		case e.TestFailed:
			suiteSummary.TestsFailed += 1
		case e.TestError:
			suiteSummary.TestsError += 1
		}
		suiteSummary.TestRuns = append(suiteSummary.TestRuns, summary)
	}

	if suiteSummary.TestsPassed == suiteSummary.TestsExecuted {
		suiteSummary.Status = SuitePassed
	} else {
		suiteSummary.Status = SuiteFailed
	}

	suiteSummary.ScenariosDuration = scenariosDuration
	suiteSummary.TotalDuration = float32(time.Since(suiteStartTime).Milliseconds())

	return suiteSummary, nil
}

// Execute a test
func (t *K6TestExecutor) execTest(
	ctx context.Context,
	testFile string,
	scenarioName string,
	env map[string]string,
) (K6TestRun, error) {
	jsonFile := getJsonOutputFilename(testFile)

	// build the command with buffer
	cmd, buf := t.prepareK6Command(
		testFile,
		jsonFile,
		env,
	)

	// run command
	var (
		cmdErr   string
		exitCode int
		status   e.TestStatus = e.TestPassed
	)
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()

			switch exitCode {
			case ThresholdFailed:
				status = e.TestFailed
			default:
				status = e.TestError
			}
		}
		cmdErr = "error running k6 command: " + err.Error()
	}

	// scenario + testDuration will be in milliseconds
	duration, err := parseDurationFromJsonFile(scenarioName, jsonFile)
	if err != nil {
		return K6TestRun{}, fmt.Errorf("error processing json file %w", err)
	}

	var (
		cloudId  string
		cloudURL string
	)
	if t.CloudOutput {
		cloudId, cloudURL, err = parseK6CloudIdentifiersFromCLIOutput(buf.Bytes())
		if err != nil {
			return K6TestRun{}, fmt.Errorf("error parsing cloud run from K6 summary %w", err)
		}
	}

	iterations, err := parseIterationCountFromCLIOutput(buf.Bytes())
	if err != nil {
		return K6TestRun{}, fmt.Errorf("error parsing iterations from k6 summary %w", err)
	}

	return K6TestRun{
		Status:      status,
		ExitCode:    exitCode,
		Durations:   duration,
		Iterations:  iterations,
		ExitMessage: cmdErr,
		CloudID:     cloudId,
		CloudURL:    cloudURL,
	}, nil
}

// pattern to match k6 version output
var k6VersionPatter = regexp.MustCompile(`v([0-9]+)(\.[0-9]+)?(\.[0-9]+)?`)

// GetK6Version checks the version of the k6 binary installed locally, returns error if none is installed
func (t K6TestExecutor) getK6Version() (string, error) {
	stdout := bytes.Buffer{}
	k6Cmd := exec.Command("k6", "version")
	k6Cmd.Stdout = &stdout

	err := k6Cmd.Run()
	if err != nil {
		return "", err
	}

	k6Version := k6VersionPatter.Find(stdout.Bytes())
	if len(k6Version) == 0 {
		return "", fmt.Errorf("could not determine k6 version")
	}
	return string(k6Version), nil
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

// getTestFiles returns the list of tests to execute.
// If tests points to a file with a js extension run that single file.
// If it points to a directory all of the .js files in it are recursively searched.
// tests=dashboard_read.js will run dashboard_read.js.
// tests=dashboards will run all files in dashboards/**.*.js.
func (t *K6TestExecutor) getTestFiles(suite e.TestSuite) ([]string, error) {
	if filepath.IsAbs(suite.Path) {
		return nil, fmt.Errorf("test suite must be a relative to base dir. Got %q", suite.Path)
	}

	testSuitePath, err := filepath.Abs(path.Join(suite.BaseDir, suite.Path))
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

// prepareK6Command builds the command with output set to standard output and a
// buffer and passes the cmd and buffer back to be executed and parsed
func (t *K6TestExecutor) prepareK6Command(testFile, jsonFile string, env map[string]string) (*exec.Cmd, *bytes.Buffer) {
	args := []string{
		"run",
		testFile,
		"--out", fmt.Sprintf(`json=%s`, jsonFile),
	}

	env["path"] = os.Getenv("PATH")
	env["K6_BROWSER_ENABLED"] = "true"
	env["K6_BROWSER_ARGS"] = "no-sandbox"

	if t.CloudOutput {
		env["K6_CLOUD_TOKEN"] = t.CloudToken
		env["K6_CLOUD_PROJECT_ID"] = t.CloudProjectID
		env["K6_CLOUD_TRACES_ENABLED"] = "true"

		args = append(args, "--out", "cloud")
	} else {
		t.Log.Warn("running load tests with cloud output disabled.")
	}

	cmd := exec.Command("k6", args...)

	// set env vars
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strings.TrimSpace(value)))
	}

	buf := bytes.NewBuffer(nil)
	if t.Verbose {
		cmd.Stdout = io.MultiWriter(buf, os.Stderr)
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = buf
	}

	return cmd, buf
}

// dashboard_create.js -> /tmp/dashboard_create.json
func getJsonOutputFilename(filename string) string {
	jsonName := filepath.Base(filename)
	jsonName = strings.TrimSuffix(jsonName, filepath.Ext(jsonName))
	return path.Join("/tmp", jsonName+".json")
}
