package executor

import (
	"context"
	"time"

	"github.com/grafana/grafana-bench/pkg/metrics"
)

type TestStatus string

const (
	Unknown     TestStatus = ""
	TestPassed  TestStatus = "passed"
	TestFlaky   TestStatus = "flaky"
	TestFailed  TestStatus = "failed"
	TestError   TestStatus = "error"
	TestSkipped TestStatus = "skipped"
)

type SuiteStatus string

const (
	SuitePassed SuiteStatus = "passed"
	SuiteFailed SuiteStatus = "failed"
)

type SuiteRun struct {
	Name           string
	Id             string
	TestExecutor   string
	Trigger        string
	SuiteName      string
	SuiteRevision  string
	GrafanaVersion string
	GrafanaURL     string
	GrafanaSlug    string
	BenchRevision  string
	Attributes     map[string]string `json:"attributes"`
}

// TestRunSummary summarizes the execution of a test
type TestRunSummary struct {
	TestFolder       string        `json:"testFolder"`
	TestFile         string        `json:"testFile"`
	StartTime        time.Time     `json:"startTime"`
	Status           TestStatus    `json:"status"`
	ExitMessage      string        `json:"exitMessage"`
	Iterations       string        `json:"iterations"`
	TotalDuration    time.Duration `json:"totalDuration"`
	ScenarioDuration time.Duration `json:"scenarioDuration"`
	// Attributes are provided by the test runner and not user configurable
	Attributes map[string]string `json:"attributes"`
}

// TestSuiteSummary summarizes the execution of  a test suite
type SuiteRunSummary struct {
	StartTime         time.Time
	Status            SuiteStatus
	TestsExecuted     int32
	TestsFailed       int32
	TestsFlaky        int32
	TestsPassed       int32
	TestsError        int32
	TotalDuration     time.Duration
	ScenariosDuration time.Duration
	TestRuns          []TestRunSummary
	Metrics           []metrics.Metric
}

// TestSuite defines the test suite
type TestSuite struct {
	Name    string
	BaseDir string
	// Path to the test suite, relative to BaseDir
	Path     string
	Revision string
}

// TestExecutor defines the methods for running a test suite
type TestExecutor interface {
	// Name returns the name of the executor
	Name() string

	// ExecTestSuite executes a test suite an reports the results
	ExecTestSuite(
		ctx context.Context,
		suite TestSuite,
		env map[string]string,
	) (SuiteRunSummary, error)
}
