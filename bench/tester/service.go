package tester

import (
	"context"
	"path"
)

type TesterService struct {
	LocalDir string

	// location of the test suite in the workdir
	TestSuiteDir string

	// location to output test results
	SummaryDir string
}

func NewTester(ctx context.Context, localDir string) *TesterService {
	return &TesterService{
		LocalDir:     localDir,
		TestSuiteDir: path.Join(localDir, "suite"),
		SummaryDir:   path.Join(localDir, "results"),
	}
}

func (t *TesterService) New(ctx context.Context, suiteRevision, tests string) (*TestRun, error) {
	return &TestRun{
		TesterService: t,
		SuiteRevision: suiteRevision,
		Tests:         tests,
	}, nil
}
