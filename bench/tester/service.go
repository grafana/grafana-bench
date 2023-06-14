package tester

import (
	"context"
	"fmt"
	"os"

	"github.com/grafana/grafana-bench/bench/provisioner"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
)

type TesterService struct {
	LocalDir     string
	testSuiteDir string
}

func NewTester(ctx context.Context, localDir string) *TesterService {
	return &TesterService{
		LocalDir: localDir,
	}
}

func (t *TesterService) New(ctx context.Context, ps *provisioner.ProvisionState) (*TestRun, error) {
	return nil, nil
}

// Resolves test suite. Always updates to latest version
// TODO: it might make sense to clone this for each build in the future, but for
// now we're just going to linnk the build suite to the service.
func (ts *TesterService) ResolveTestSuite() error {
	exists, _ := utils.PathExists(ts.testSuiteDir)

	// If exist, update to latest
	if exists {
		err := utils.DoInDir(ts.LocalDir, ts.testSuiteDir, func() error {
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
	if err := os.MkdirAll(ts.testSuiteDir, 0755); err != nil {
		return fmt.Errorf("test-service: could not clone build suite: %w", err)
	}

	// clone path to dir
	fmt.Println("test-service: cloning build suite")
	if err := sh.RunV("git", "clone", "https://github.com/grafana/grafana-api-tests", ts.testSuiteDir); err != nil {
		return fmt.Errorf("Error checking out grafana test repo %s", err)
	}

	return nil
}
