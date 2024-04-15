package cypress

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils"
)

var (
	errMissingRepo           = errors.New("missing test suite repository")
	errMissingTargetDirError = errors.New("missing target directory to clone repository")
)

// CypressTestExecutor implements TestExecutor interface for running k6 test suites
type CypressTestExecutor struct {
	Log     *slog.Logger
	Verbose bool

	TargetDir         string
	TestSuiteRepo     string
	TestSuiteRevision string
}

// NewCypressTestExecutor creates a new instance of CypressTestExecutor
func NewCypressTestExecutor(
	log *slog.Logger,
	verbose bool,
	testSuiteRepo string,
	targetDir string,
) *CypressTestExecutor {
	return &CypressTestExecutor{
		Log:           log,
		Verbose:       verbose,
		TestSuiteRepo: testSuiteRepo,
		TargetDir:     targetDir,
	}
}

func (t *CypressTestExecutor) Name() string {
	return "Cypress"
}

// go run . test test --test-suite /path/to/test/folder --test-type smoke --runner cypress --test-dir "./test-repo" --test-suite-repo git@github.com:grafana/plugins-private
// execute test suite
func (t *CypressTestExecutor) ExecTestSuite(
	ctx context.Context,
	suite executor.TestSuite,
	env map[string]string,
) (executor.SuiteRunSummary, error) {

	if t.TestSuiteRepo == "" {
		return executor.SuiteRunSummary{}, errMissingRepo
	}

	if t.TargetDir == "" {
		return executor.SuiteRunSummary{}, errMissingTargetDirError
	}

	testingDir := utils.GetTestingDirectory(t.TargetDir, t.TestSuiteRepo)
	workingDir := "e2e"

	err := utils.ImportSetupRepo(testingDir, t.TestSuiteRepo, t.Log)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to import repo: %s", err.Error())
	}

	err = utils.ExecuteInDir(testingDir+"/"+workingDir, func() error {
		// idea: add a config in the repo with setup instructions
		installCmd := exec.Command("yarn", "install")
		if err := utils.ExecStdout(installCmd); err != nil {
			return err
		}

		executeCmd := exec.Command("yarn", "e2e")
		if err := utils.ExecStdout(executeCmd); err != nil {
			return err
		}

		return nil
	})

	// process might return exit code 1 but we still want to try to parse the report
	// if err != nil {
	// 	t.Log.Info("Cypress processes exited with code 1", "error", err.Error())
	// }

	file, err := os.ReadFile(fmt.Sprintf("%s/Cypress-report/report.json", t.TargetDir))
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to read report.json: %s", err.Error())
	}

	runSummary, err := parseJsonOutput(file)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed parsing Cypress report.json into summary: %s", err.Error())
	}

	return runSummary, nil
}
