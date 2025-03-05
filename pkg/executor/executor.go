package executor

import (
	"context"
	"time"
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

type TestDurations struct {
	SetupDuration    float32
	ScenarioDuration float32
	TeardownDuration float32
	TotalDuration    float32
}

type SuiteStatus string

const (
	SuitePassed SuiteStatus = "passed"
	SuiteFailed SuiteStatus = "failed"
)

// TestRun summarizes the execution of a test
type TestRun struct {
	TestFolder  string
	TestFile    string
	StartTime   time.Time
	Status      TestStatus
	ExitMessage string
	Iterations  string
	Durations   TestDurations
	Attributes  map[string]string
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
	TotalDuration     float32
	ScenariosDuration float32
	TestRuns          []TestRun
	Metrics           map[string]string
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
