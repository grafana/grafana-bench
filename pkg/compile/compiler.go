package compile

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/grafana/grafana-bench/pkg/git"
	"github.com/grafana/grafana-bench/pkg/git/gogit"
	"github.com/grafana/grafana-bench/pkg/git/nanogit"
	"github.com/grafana/grafana-bench/pkg/utils"
)

// TestCompiler
type TestCompiler struct {
	Log               *slog.Logger
	Driver            string
	TargetDir         string
	TestSuiteRepo     string
	CheckoutDirs      []string
	TestSuiteRevision string
	RepoToken         string
	TestPrepareCmd    []string
}

func NewTestCompiler(
	log *slog.Logger,
	driver string,
	targetDir string,
	testSuiteRepo string,
	checkOutDirs []string,
	repoToken string,
	testSuiteRevision string,
	testPrepareCmd []string,
) *TestCompiler {
	return &TestCompiler{
		Log:               log,
		Driver:            driver,
		TargetDir:         targetDir,
		TestSuiteRepo:     testSuiteRepo,
		CheckoutDirs:      checkOutDirs,
		RepoToken:         repoToken,
		TestSuiteRevision: testSuiteRevision,
		TestPrepareCmd:    testPrepareCmd,
	}
}

// CompileTestSuite collect the test suite from a source repository
// returns the test suite revision
func (tc *TestCompiler) CompileTestSuite(ctx context.Context) (string, error) {
	var (
		gitSource git.GitSource
		err       error
	)

	switch tc.Driver {
	case "nanogit":
		gitSource, err = nanogit.NewSource(tc.TestSuiteRepo, tc.RepoToken)
	case "gogit":
		gitSource, err = gogit.NewSource(tc.TestSuiteRepo, tc.RepoToken)
	default:
		return "", fmt.Errorf("unknown git driver %s", tc.Driver)
	}

	if err != nil {
		return "", fmt.Errorf("creating git source %s: %w", tc.TestSuiteRepo, err)
	}

	revision, err := gitSource.Get(ctx, tc.TargetDir, tc.TestSuiteRevision, tc.CheckoutDirs...)
	if err != nil {
		return "", fmt.Errorf("checking out test suite %s: %w", tc.TestSuiteRepo, err)
	}

	// build the tests
	if len(tc.TestPrepareCmd) > 0 {
		workDir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting current work directory %w", err)
		}

		err = utils.DoInDir(workDir, tc.TargetDir, func() error {
			cmdMake := exec.Command(tc.TestPrepareCmd[0], tc.TestPrepareCmd[1:]...)
			if err := utils.ExecStdout(cmdMake); err != nil {
				return fmt.Errorf("building test suite: %w", err)
			}

			return nil
		})

		return "", err
	}

	return revision, nil
}
