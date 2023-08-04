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

	// TODO implement me
	// if enabled, always clone the test suite into a separate folder to ensure
	// that the state of the folder does not clash when running multiple bench marks at the
	// same time
	AlwaysCloneTestSuite bool

	// k6CloudToken
	K6CloudToken     string
	K6CloudProjectId string

	// location of the test results
	resultsDir string
}

func NewTester(ctx context.Context, localDir, resultsDir, k6CloudToken, k6CloudProjectId string) *TesterService {
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
		LocalDir:         localDir,
		TestSuiteDir:     testSuiteDir,
		resultsDir:       resultsDir,
		K6CloudToken:     k6CloudToken,
		K6CloudProjectId: k6CloudProjectId,
	}
}

// New creates a new test run.
// suiteREvision is optional. If left blank, it will use exactly what is in the
// test repo for local development. Otherwise, provide branch or commit.
//
// tests takes a foloder or .js file in the tests/ directory of
// https://github.com/grafana/grafana-api-tests
//
// reportToK6Cloud sends results to k6 cloud if true

func (t *TesterService) New(ctx context.Context, suiteRevision, tests string, reportToK6Cloud bool) (*TestRun, error) {
	return &TestRun{
		TesterService:   t,
		SuiteRevision:   suiteRevision,
		Tests:           tests,
		ReportToK6Cloud: reportToK6Cloud,
	}, nil
}
