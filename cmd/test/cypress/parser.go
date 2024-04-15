package cypress

import (
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// parseJsonOutput parses the json output from playwright --report json and returns a slice of RunSummary
// this will work if only one test is run and the output but will also work for if this contains an entire suite
func parseJsonOutput(buf []byte) (executor.SuiteRunSummary, error) {
	output := CypressJsonOutput{}

	err := json.Unmarshal(buf, &output)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("parsing Playwright json summary output: %w", err)
	}

	// Convert Unix timestamps to time.Time
	start := time.Unix(output.Results.Summary.Start, 0)
	end := time.Unix(output.Results.Summary.Stop, 0)
	duration := end.Sub(start)

	testRuns := make([]executor.TestRun, 0, output.Results.Summary.Tests)
	testStartTime := start
	for order, test := range output.Results.Tests {
		run := formatTestRuns(test, testStartTime, order)
		testRuns = append(testRuns, run)

		testStartTime = testStartTime.Add(time.Duration(test.Duration) * time.Millisecond)
	}

	suiteRunSummary := executor.SuiteRunSummary{
		StartTime:         start,
		ScenariosDuration: float32(duration.Milliseconds()),
		TestsExecuted:     int32(output.Results.Summary.Tests),
		TestsFailed:       int32(output.Results.Summary.Failed),
		TestsPassed:       int32(output.Results.Summary.Passed),
		TestsError:        int32(output.Results.Summary.Failed),
		TotalDuration:     float32(duration.Milliseconds()),
		TestRuns:          testRuns,
	}

	return suiteRunSummary, nil
}

func formatTestRuns(test Tests, startTime time.Time, order int) executor.TestRun {

	testStatus := executor.TestPassed
	exitCode := 0
	if test.Status == "failed" {
		testStatus = executor.TestFailed
		exitCode = 1
	}

	summary := executor.TestRun{
		TestFolder: test.Name,
		TestFile:   path.Base(test.FilePath),

		StartTime: startTime,

		Status:      testStatus,
		ExitMessage: test.Message,

		Order:    order,
		ExitCode: exitCode,

		Iterations: string(test.Retry),

		Durations: executor.TestDurations{
			SetupDuration:    0,
			TeardownDuration: 0,
			ScenarioDuration: float32(test.Duration),
			TotalDuration:    float32(test.Duration),
		},
	}

	return summary
}
