package compile

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

// TestCompiler
type TestCompiler struct {
	Log               *slog.Logger
	BaseDir           string
	TestSuiteRepo     string
	TestSuiteRevision string
}


func NewTestCompiler(
	log *slog.Logger,
	baseDir string,
	testSuiteRepo string,
	testSuiteRevision string,
)  *TestCompiler {
	return &TestCompiler{
		Log:               log,
		BaseDir:           baseDir,
		TestSuiteRepo:     testSuiteRepo,
		TestSuiteRevision: testSuiteRevision,
	}
}

// PackTests collects and builds tests from a source repository
func (tc *TestCompiler)CompileTestSuite(ctx context.Context) error {
	// clone repo if doesn't exist
	exists, _ := utils.PathExists(tc.BaseDir)
	if !exists {
		tc.Log.Info("cloning build suite")

		// TODO remove dependency on mage.sh.RunV
		if err := sh.RunV("git", "clone", tc.TestSuiteRepo, tc.BaseDir); err != nil {
			return fmt.Errorf("checking out test repo %s: %w", tc.TestSuiteRepo, err)
		}
	}

	// update repo + checkout branch
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current work directory %w", err)
	}

	err = utils.DoInDir(workDir, tc.BaseDir, func() error {
		// if we don't specify a revision, assume we want to run exactly what is
		// there. e.g. local development
		if tc.TestSuiteRevision != "" {
			// check current branch. Don't do anything if it's the same as what is
			// currently there.
			// TODO: this logic may not make sense in scenarios where it's set to main
			// but someone hasn't checked out in a while and wants to update. That would
			// require a manual update. perhaps check the git sha. review later.
			cmdBranch := exec.Command("git", "branch", "--show-current")
			branch, err2 := cmdBranch.CombinedOutput()
			if err2 != nil {
				return fmt.Errorf("error getting current branch %w", err2)
			}
			if strings.TrimSpace(string(branch)) == tc.TestSuiteRevision {
				return nil
			}

			// get latest main
			if err := utils.ExecStdout(exec.Command("git", "checkout", "main")); err != nil {
				return fmt.Errorf("checking out main branch from repo %s", err)
			}

			// update
			if err := utils.ExecStdout(exec.Command("git", "pull")); err != nil {
				return err
			}

			// checkout if not main
			if tc.TestSuiteRevision != "main" {
				cmdCheckout := exec.Command("git", "checkout", tc.TestSuiteRevision)
				if err := utils.ExecStdout(cmdCheckout); err != nil {
					return err
				}
			}
		}

		// build the tests
		cmdMake := exec.Command("make", "build")
		if err := utils.ExecStdout(cmdMake); err != nil {
			return fmt.Errorf("Error building test suite: %w", err)
		}

		return nil
	})

	return err
}


