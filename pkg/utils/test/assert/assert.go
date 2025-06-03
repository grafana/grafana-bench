package assert

import (
	"fmt"
	"sort"

	"github.com/grafana/grafana-bench/pkg/executor"
)

func sortTestRunByFilename(tr []executor.TestRunSummary) {
	sort.Slice(tr, func(i, j int) bool {
		return tr[i].TestFile < tr[j].TestFile
	})
}

func newAssertionError(message string, expected, actual any) error {
	return fmt.Errorf("%s: expected: %v got: %v", message, expected, actual)
}

func TestRunEqual(expected *executor.TestRunSummary, actual executor.TestRunSummary) error {
	if expected == nil {
		return nil
	}

	if expected.TestFile != actual.TestFile {
		return newAssertionError("test filename assertion failed", expected.TestFile, actual.TestFile)
	}

	if expected.Status != actual.Status {
		return newAssertionError("test status assertion failed", expected.Status, actual.Status)
	}

	return nil
}

func SuiteSummaryEqual(expected *executor.SuiteRunSummary, actual executor.SuiteRunSummary) error {
	if expected == nil {
		return nil
	}

	if expected.TestsExecuted != actual.TestsExecuted {
		return newAssertionError("test executed assertion failed", expected.TestsExecuted, actual.TestsExecuted)
	}

	if expected.TestsError != actual.TestsError {
		return newAssertionError("test error assertion failed", expected.TestsError, actual.TestsError)
	}

	if expected.TestsFailed != actual.TestsFailed {
		return newAssertionError("test failed assertion failed", expected.TestsFailed, actual.TestsFailed)
	}

	if expected.TestsPassed != actual.TestsPassed {
		return newAssertionError("test passed assertion failed", expected.TestsPassed, actual.TestsPassed)
	}

	if len(expected.TestRuns) != len(actual.TestRuns) {
		return newAssertionError("number of test runs assertion failed", len(expected.TestRuns), len(actual.TestRuns))
	}

	sortTestRunByFilename(expected.TestRuns)
	sortTestRunByFilename(actual.TestRuns)

	for i, tr := range expected.TestRuns {
		if err := TestRunEqual(&tr, actual.TestRuns[i]); err != nil {
			return err
		}
	}

	return nil
}
