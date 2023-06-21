package tester

import (
	"context"
	"fmt"
	"os"
	"path"
)

// 1. move test repo into uuid dir
// 2. move results into results directory
// 3. upload results to bucket

type TesterService struct {
	LocalDir string

	// location of the test suite in the workdir
	TestSuiteDir string

	// location of the test results
	resultsDir string
}

func NewTester(ctx context.Context, localDir, resultsDir string) *TesterService {
	testSuiteDir := path.Join(localDir, "suite")

	err := os.MkdirAll(testSuiteDir, 0755)
	if err != nil {
		panic(fmt.Errorf("tester: error creating test suite directory: %w", err))
	}

	err = os.MkdirAll(resultsDir, 0755)
	if err != nil {
		panic(fmt.Errorf("tester: error creating test suite directory: %w", err))
	}

	return &TesterService{
		LocalDir:     localDir,
		TestSuiteDir: testSuiteDir,
		resultsDir:   resultsDir,
	}
}

func (t *TesterService) New(ctx context.Context, suiteRevision, tests string) (*TestRun, error) {
	return &TestRun{
		TesterService: t,
		SuiteRevision: suiteRevision,
		Tests:         tests,
	}, nil
}
