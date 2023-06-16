package tester

import (
	"context"
	"fmt"
	"os"

	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

type TesterService struct {
	LocalDir string

	// location of the test suite in the workdir
	TestSuiteDir string
}

func NewTester(ctx context.Context, localDir string) *TesterService {
	return &TesterService{
		LocalDir: localDir,
	}
}

func (t *TesterService) New(ctx context.Context) (*TestRun, error) {
	defaultTestSuite := "dashboards"

	return &TestRun{
		TesterService: t,
		testSuite:     defaultTestSuite,
	}, nil
}

// Resolves test suite. Always updates to latest version
// TODO: it might make sense to clone this for each build in the future, but for
// now we're just going to linnk the build suite to the service.
func (ts *TesterService) ResolveTestSuite() error {
	exists, _ := utils.PathExists(ts.LocalDir)

	// If exist, update to latest
	if exists {
		err := utils.DoInDir(ts.LocalDir, ts.LocalDir, func() error {
			if err := sh.RunV("git", "checkout", "main"); err != nil {
				return fmt.Errorf("test-service: Error checking out grafana test repo %s", err)
			}

			if err := sh.RunV("git", "pull"); err != nil {
				return err
			}

			return nil
		})
		return err
	}

	// ensure path exists
	if err := os.MkdirAll(ts.LocalDir, 0755); err != nil {
		return fmt.Errorf("test-service: could not clone build suite: %w", err)
	}

	// clone path to dir
	fmt.Println("test-service: cloning build suite")
	if err := sh.RunV("git", "clone", "https://github.com/grafana/grafana-api-tests", ts.LocalDir); err != nil {
		return fmt.Errorf("Error checking out grafana test repo %s", err)
	}

	return nil
}
