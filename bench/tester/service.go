package tester

import (
	"context"
	"fmt"
	"os"
	"path"
)

type TesterService struct {
	LocalDir string

	// location of the test suite in the workdir
	TestSuiteDir string

	// k6CloudToken
	K6CloudToken string

	// location of the test results
	resultsDir string
}

func NewTester(ctx context.Context, localDir, resultsDir, k6CloudToken string) *TesterService {
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
		K6CloudToken: k6CloudToken,
	}
}

func (t *TesterService) New(ctx context.Context, suiteRevision, tests string, reportToK6Cloud bool) (*TestRun, error) {
	return &TestRun{
		TesterService:   t,
		SuiteRevision:   suiteRevision,
		Tests:           tests,
		ReportToK6Cloud: reportToK6Cloud,
	}, nil
}
