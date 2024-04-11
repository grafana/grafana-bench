package test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/grafana/grafana-bench/pkg/utils"
)

const (
// ThresholdFailed = 99 // return code when test thresholds fail
)

var (
	errMissingRepo           = errors.New("missing test suite repository")
	errMissingTargetDirError = errors.New("missing target directory to clone repository")
)

// PlaywrightTestExecutor implements TestExecutor interface for running k6 test suites
type PlaywrightTestExecutor struct {
	Log     *slog.Logger
	Verbose bool

	TargetDir         string
	TestSuiteRepo     string
	TestSuiteRevision string
}

// NewPlaywrightTestExecutor creates a new instance of PlaywrightTestExecutor
func NewPlaywrightTestExecutor(
	log *slog.Logger,
	verbose bool,
	testSuiteRepo string,
	targetDir string,
) *PlaywrightTestExecutor {
	return &PlaywrightTestExecutor{
		Log:           log,
		Verbose:       verbose,
		TestSuiteRepo: testSuiteRepo,
		TargetDir:     targetDir,
	}
}

// K6TestRun summarizes the execution of a k6 test
type PlaywrightTestRun struct {
	// Status      TestStatus
	ExitCode    int
	ExitMessage string
	Iterations  string
	// Durations   TestDurations
	CloudID  string
	CloudURL string
}

func (t *PlaywrightTestExecutor) Name() string {
	return "playwright"
}

// execute test suite
func (t *PlaywrightTestExecutor) ExecTestSuite(
	ctx context.Context,
	suite TestSuite,
	env map[string]string,
) (SuiteRunSummary, error) {

	// 1. get url for repo
	t.Log.Debug("TestSuiteRepo", t.TestSuiteRepo)
	if t.TestSuiteRepo == "" {
		return SuiteRunSummary{}, errMissingRepo
	}

	t.Log.Debug("TargetDir", t.TargetDir)
	if t.TargetDir == "" {
		return SuiteRunSummary{}, errMissingTargetDirError
	}

	// 2. download and setup repo
	t.Log.Debug("TestSuiteRepo", t.TestSuiteRepo, "TargetDir", t.TargetDir)
	err := ImportSetupRepo(t.TargetDir, t.TestSuiteRepo, t.Log)
	if err != nil {
		return SuiteRunSummary{}, fmt.Errorf("failed to import repo: %s", err.Error())
	}

	// // 3. run tests
	workDir, err := os.Getwd()
	if err != nil {
		return SuiteRunSummary{}, err
	}
	err = utils.DoInDir(workDir, t.TargetDir, func() error {
		// add a config in the repo with setup instructions
		installCmd := exec.Command("yarn", "test")
		if err := utils.ExecStdout(installCmd); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return SuiteRunSummary{}, err
	}

	file, err := os.ReadFile(fmt.Sprintf("%s/playwright-report/report.json", t.TargetDir))
	if err != nil {
		return SuiteRunSummary{}, fmt.Errorf("failed to read report.json: %s", err.Error())

	}

	println("File", string(file))

	return SuiteRunSummary{}, nil
}
