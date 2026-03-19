package k6

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
	"slices"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	k6parser "github.com/grafana/grafana-bench/pkg/parser/k6"
	"github.com/grafana/grafana-bench/pkg/utils"
	"github.com/grafana/grafana-bench/pkg/utils/format"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/grafana/k6pack"
)

const (
	ThresholdFailed = 99 // return code when test thresholds fail
)

var (
	missingK6CloudConfigError = errors.New("k6 Token and project ID are required for cloud output")
	testFilesError            = errors.New("getting test files")
	testExts                  = []string{".js", ".ts"}
	// used to replace strings.Title
	caser = cases.Title(language.AmericanEnglish)
)

// K6TestExecutor implements TestExecutor interface for running k6 test suites
type K6TestExecutor struct {
	Log *slog.Logger
	K6ExecutorOptions
}

type K6ExecutorOptions struct {
	RetryFailed    int
	Verbose        bool
	CloudOutput    bool
	CloudToken     string
	CloudProjectID string
}

// NewK6TestExecutor creates a new instance of K6TestExecutor
func NewK6TestExecutor(
	log *slog.Logger,
	opts K6ExecutorOptions,
) *K6TestExecutor {
	return &K6TestExecutor{
		Log:               log.With("executor", "k6"),
		K6ExecutorOptions: opts,
	}
}

type k6Output struct {
	iterations string
	durations  k6parser.TestDurations
	cloudId    string
	cloudURL   string
}

// K6TestRun summarizes the execution of a k6 test
type K6TestRun struct {
	Status      executor.TestStatus
	ExitCode    int
	ExitMessage string
	Iterations  string
	Durations   k6parser.TestDurations
	CloudID     string
	CloudURL    string
}

func (t *K6TestExecutor) Name() string {
	return "k6"
}

// execute test suite
func (t *K6TestExecutor) ExecTestSuite(
	ctx context.Context,
	suite executor.TestSuite,
	env map[string]string,
) (executor.SuiteRunSummary, error) {
	if t.CloudOutput && (t.CloudToken == "" || t.CloudProjectID == "") {
		return executor.SuiteRunSummary{}, missingK6CloudConfigError
	}

	// set common test execution variables
	k6env := map[string]string{}

	// copy environment variables passed to execution
	for k, v := range env {
		k6env[k] = v
	}

	// run k6 tests
	var (
		scenariosDuration time.Duration
	)

	k6Version, err := t.getK6Version()
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("getting k6 version %w", err)
	}

	t.Log.Debug("using k6", "k6Version", k6Version)

	tests, err := t.getTestFiles(suite)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("%w: %w", testFilesError, err)
	}

	suiteSummary := executor.SuiteRunSummary{}
	suiteSummary.StartTime = time.Now()
	suiteSummary.SuiteName = suite.Name
	suiteSummary.SuiteRevision = suite.Revision

	// run the tests
	for _, testFile := range tests {
		scenarioName := getScenarioName(testFile)
		k6env["SCENARIO_NAME"] = scenarioName

		var (
			testStartTime time.Time
			retries       int
			k6Summary     K6TestRun
		)

		for {
			// reset the start time for each test retry
			testStartTime = time.Now()

			// run command send output to cloud
			k6Summary, err = t.execTest(
				ctx,
				testFile,
				scenarioName,
				k6env,
			)
			if err != nil {
				t.Log.Error("executing k6 test", "error", err.Error())
				// TODO: maybe we should break the iteration here, as test result may not be relevant
			}

			if k6Summary.Status != executor.TestFailed || retries == t.RetryFailed {
				break
			}
			retries++
		}

		if k6Summary.Status == executor.TestPassed && retries > 0 {
			k6Summary.Status = executor.TestFlaky
		}

		scenariosDuration += k6Summary.Durations.TotalDuration

		// get the path to the test relative to the TestSuiteBase if any
		// we don't need to check for errors because how the test path is constructed
		rootDir, _ := filepath.Abs(suite.BaseDir)
		testFolder, _ := filepath.Rel(rootDir, filepath.Dir(testFile))

		summary := executor.TestRunSummary{
			TestFolder:       testFolder,
			TestFile:         path.Base(testFile),
			StartTime:        testStartTime,
			Status:           k6Summary.Status,
			TotalDuration:    k6Summary.Durations.TotalDuration,
			ScenarioDuration: k6Summary.Durations.ScenarioDuration,
			Iterations:       k6Summary.Iterations,
			ExitMessage:      k6Summary.ExitMessage,
			Attributes: map[string]string{
				"cloudId":          k6Summary.CloudID,
				"cloudURL":         k6Summary.CloudURL,
				"setupDuration":    format.PrettyMS(k6Summary.Durations.SetupDuration),
				"teardownDuration": format.PrettyMS(k6Summary.Durations.TeardownDuration),
			},
		}

		suiteSummary.TestsExecuted += 1
		switch summary.Status {
		case executor.TestPassed:
			suiteSummary.TestsPassed += 1
		case executor.TestFlaky:
			suiteSummary.TestsFlaky += 1
		case executor.TestFailed:
			suiteSummary.TestsFailed += 1
		case executor.TestError:
			suiteSummary.TestsError += 1
		}
		suiteSummary.TestRuns = append(suiteSummary.TestRuns, summary)
	}

	if suiteSummary.TestsFailed+suiteSummary.TestsError == 0 {
		suiteSummary.Status = executor.SuitePassed
	} else {
		suiteSummary.Status = executor.SuiteFailed
	}

	suiteSummary.ScenariosDuration = scenariosDuration
	suiteSummary.TotalDuration = time.Since(suiteSummary.StartTime)

	return suiteSummary, nil
}

// Execute a test
func (t *K6TestExecutor) execTest(
	ctx context.Context,
	testFile string,
	scenarioName string,
	env map[string]string,
) (K6TestRun, error) {
	testFile, err := transpileTest(testFile)
	if err != nil {
		return K6TestRun{}, err
	}

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
		status   executor.TestStatus = executor.TestPassed
		output   k6Output
	)
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()

			switch exitCode {
			case ThresholdFailed:
				status = executor.TestFailed
			default:
				status = executor.TestError
			}
		}
		cmdErr = "error running k6 command: " + err.Error()
		t.Log.Error(cmdErr)
		// avoid duplicating outout in verbose mode
		if !t.Verbose {
			fmt.Println(buf.String())
		}
	}

	if status != executor.TestError {
		output, err = t.getOutput(buf, jsonFile, scenarioName)
	}

	return K6TestRun{
		Status:      status,
		ExitCode:    exitCode,
		Durations:   output.durations,
		Iterations:  output.iterations,
		ExitMessage: cmdErr,
		CloudID:     output.cloudId,
		CloudURL:    output.cloudURL,
	}, err
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

func transpileTest(testFile string) (string, error) {
	source, err := os.ReadFile(testFile)
	if err != nil {
		return "", fmt.Errorf("reading test file %q: %w", testFile, err)
	}

	transpiled, err := os.CreateTemp("", filepath.Base(testFile)+".js")
	if err != nil {
		return "", fmt.Errorf("creating temp test file %q: %w", testFile, err)
	}

	buf, _, err := k6pack.Pack(string(source), &k6pack.Options{
		Filename:   testFile,
		TypeScript: true,
		SourceMap:  true,
		SourceRoot: filepath.Dir(testFile),
	})
	if err != nil {
		return "", fmt.Errorf("transpiling test file %q: %w", testFile, err)
	}

	_, err = io.Copy(transpiled, bytes.NewBuffer(buf))
	if err != nil {
		return "", fmt.Errorf("copying temp test file %q: %w", testFile, err)
	}

	return transpiled.Name(), nil
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
		parts[i] = caser.String(p)
	}
	return strings.Join(parts, "")
}

// getTestFiles returns the list of tests to execute.
// If tests points to a file with a js extension run that single file.
// If it points to a directory all of the .js files in it are recursively searched.
// tests=dashboard_read.js will run dashboard_read.js.
// tests=dashboards will run all files in dashboards/**.*.js.
func (t *K6TestExecutor) getTestFiles(suite executor.TestSuite) ([]string, error) {
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
		files, err := utils.GlobByExtension(testSuitePath, testExts...)
		if err != nil {
			return nil, err
		}

		if len(files) == 0 {
			return nil, fmt.Errorf("no test files found at %s", testSuitePath)
		}

		return files, nil
	}

	// is a file, we expect a single .js or .ts file
	if !slices.Contains(testExts, filepath.Ext(testSuitePath)) {
		return nil, fmt.Errorf("expected a '.js' or '.ts'. file. Got %s", testSuitePath)
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
	}

	cmd := exec.Command("k6", args...)

	// set env vars
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strings.TrimSpace(value)))
	}

	buf := bytes.NewBuffer(nil)
	if t.Verbose {
		cmd.Stdout = io.MultiWriter(buf, os.Stderr)
		cmd.Stderr = io.MultiWriter(buf, os.Stderr)
	} else {
		cmd.Stdout = buf
		cmd.Stderr = buf
	}

	return cmd, buf
}

// parses the execution outputs and returns a summary
func (t *K6TestExecutor) getOutput(buf *bytes.Buffer, jsonFile string, scenarioName string) (k6Output, error) {
	// scenario + testDuration will be in milliseconds
	duration, err := k6parser.ParseDurationFromJsonFile(scenarioName, jsonFile)
	if err != nil {
		return k6Output{}, fmt.Errorf("error processing json file %w", err)
	}

	var (
		cloudId  string
		cloudURL string
	)
	if t.CloudOutput {
		cloudId, cloudURL, err = k6parser.ParseK6CloudIdentifiersFromCLIOutput(buf.Bytes())
		if err != nil {
			return k6Output{}, fmt.Errorf("error parsing cloud run from K6 summary %w", err)
		}
	}

	iterations, err := k6parser.ParseIterationCountFromCLIOutput(buf.Bytes())
	if err != nil {
		return k6Output{}, fmt.Errorf("error parsing iterations from k6 summary %w", err)
	}

	return k6Output{
		durations:  duration,
		iterations: iterations,
		cloudId:    cloudId,
		cloudURL:   cloudURL,
	}, err
}

// dashboard_create.js -> /tmp/dashboard_create.json
func getJsonOutputFilename(filename string) string {
	jsonName := filepath.Base(filename)
	jsonName = strings.TrimSuffix(jsonName, filepath.Ext(jsonName))
	return path.Join("/tmp", jsonName+".json")
}
