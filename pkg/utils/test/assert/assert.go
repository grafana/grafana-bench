package assert

import (
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils/test/sort"

)

func Equal(t *testing.T, message string, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s expected '%v', got '%v'", message, expected, actual)
	}
}

func TestRunEqual(t *testing.T, expected *executor.TestRunSummary, actual executor.TestRunSummary) {
	if expected == nil {
		return
	}

	Equal(t, "test filename assertion failed", expected.TestFile, actual.TestFile)
	Equal(t, "test status assertion failed", expected.Status, actual.Status)
	Equal(t, "test durations assertion failed", expected.Durations, actual.Durations)
}

func SuiteSummaryEqual(t *testing.T, expected *executor.SuiteRunSummary, actual executor.SuiteRunSummary) {
	if expected == nil {
		return
	}

	Equal(t, "test executed assertion failed", expected.TestsExecuted, actual.TestsExecuted)
	Equal(t, "test error assertion failed", expected.TestsError, actual.TestsError)
	Equal(t, "test failed assertion failed", expected.TestsFailed, actual.TestsFailed)
	Equal(t, "test passed assertion failed", expected.TestsPassed, actual.TestsPassed)
	Equal(t, "number of test runs assertion failed", len(expected.TestRuns), len(actual.TestRuns))
	Equal(t, "total durations assertion failed", expected.TotalDuration, actual.TotalDuration)
	Equal(t, "scenario durations assertion failed", expected.ScenariosDuration, actual.ScenariosDuration)

	sort.SortTestRunByFilename(expected.TestRuns)
	sort.SortTestRunByFilename(actual.TestRuns)

	for i, tr := range expected.TestRuns {
		TestRunEqual(t, &tr, actual.TestRuns[i])
	}
}
