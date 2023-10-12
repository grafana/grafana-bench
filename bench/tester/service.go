package tester

import (
	"context"
	"fmt"
	"os"
	"path"

	"log/slog"
)

type TesterService struct {
	Log              *slog.Logger
	LocalDir         string
	UseCompiledTests bool
	// location of the test suite in the workdir
	TestSuiteDir string
	// location of tests in side the test suite
	TestRoot string
	// location of the .version file within the test repo
	VersionFilePath  string
	K6CloudToken     string
	K6CloudProjectId string
	GrafanaTestRepo  string
}

func NewTester(ctx context.Context, log *slog.Logger, localDir string, useCompiledTests bool, grafanaTestRepo, k6CloudProjectId, k6CloudToken string) *TesterService {
	log = log.With("svc", "tester")

	err := os.MkdirAll(localDir, 0755)
	if err != nil {
		panic(fmt.Errorf("tester: could not create test service working directory: %w", err))
	}

	var testSuiteDir, testRoot, versionFilePath string
	if useCompiledTests {
		// just use the same folder for everything if precompiled
		testSuiteDir = localDir
		testRoot = localDir
		versionFilePath = path.Join(localDir, ".version")
	} else {
		testSuiteDir = path.Join(localDir, "suite")
		testRoot = path.Join(testSuiteDir, "dist", "tests")
		versionFilePath = path.Join(testRoot, ".version")
	}

	return &TesterService{
		Log:              log,
		LocalDir:         localDir,
		UseCompiledTests: useCompiledTests,
		TestSuiteDir:     testSuiteDir,
		TestRoot:         testRoot,
		VersionFilePath:  versionFilePath,
		GrafanaTestRepo:  grafanaTestRepo,
		K6CloudToken:     k6CloudToken,
		K6CloudProjectId: k6CloudProjectId,
	}
}

// New creates a new test run.
// suiteRevision is optional. If left blank, it will use exactly what is in the
// test repo for local development. Otherwise, provide branch or commit.
//
// tests takes a folder or .js file in the dist/ directory of
// https://github.com/grafana/grafana-api-tests
func (t *TesterService) New(ctx context.Context, suiteRevision string, runType string, tests string) (*TestRun, error) {
	trt := TestRunTypeFromString(runType)

	return &TestRun{
		TesterService: t,
		SuiteRevision: suiteRevision,
		Type:          trt,
		Tests:         tests,
	}, nil
}
