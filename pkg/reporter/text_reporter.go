package reporter

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// TextReporter reports a test suite run using structured logs
type TextReporter struct {
	report io.Writer
}

func NewTextReporter(report io.Writer) *TextReporter {
	return &TextReporter{
		report: report,
	}
}

func (r *TextReporter) Report(
	_ context.Context,
	_ executor.SuiteRun,
	suiteRunSummary executor.SuiteRunSummary,
) error {
	tw := tabwriter.NewWriter(r.report, 5, 0, 1, ' ', 0)
	defer tw.Flush()

	for _, testRun := range suiteRunSummary.TestRuns {
		fmt.Fprintf(
			tw,
			"[%s]\t%.2f sec\t%s:\t%s\n",
			strings.ToUpper(string(testRun.Status)),
			testRun.TotalDuration.Seconds(),
			testRun.TestFolder,
			testRun.TestFile,
		)
	}

	testsByStatus := make(map[executor.TestStatus][]string, 0)
	for _, testRun := range suiteRunSummary.TestRuns {
		fmt.Fprintf(
			tw,
			"[%s]\t%.2f sec\t%s:\t%s\n",
			strings.ToUpper(string(testRun.Status)),
			testRun.TotalDuration.Seconds(),
			testRun.TestFolder,
			testRun.TestFile,
		)
	
		if testRun.Status == executor.TestPassed && testRun.Status != executor.TestSkipped {
			continue
		}
	
		// collect tests that didn't pass
		tests := testsByStatus[testRun.Status]
		tests = append( tests, fmt.Sprintf("%s:\t%s", testRun.TestFolder, testRun.TestFile))
		testsByStatus[testRun.Status] = tests
	}

	statuses := []executor.TestStatus{executor.TestError, executor.TestFailed, executor.TestFlaky}
	for _, status := range statuses {
		tests := testsByStatus[status]
		if len(tests) > 0 {
			fmt.Fprintf(tw, "\n--------------%s TESTS--------------\n", strings.ToUpper(string(status)))
			for _, test := range tests {
				fmt.Fprintf(tw, "%s\n", test)
			}
		}
	}

	fmt.Fprintf(tw, "\n----------------SUMMARY----------------\n")
	fmt.Fprintf(tw, "Executed:\t%d\n", suiteRunSummary.TestsExecuted)
	fmt.Fprintf(tw, "Passed:\t%d\n", suiteRunSummary.TestsPassed)
	fmt.Fprintf(tw, "Flaky:\t%d\n", suiteRunSummary.TestsFlaky)
	fmt.Fprintf(tw, "Failed:\t%d\n", suiteRunSummary.TestsFailed)
	fmt.Fprintf(tw, "Errors:\t%d\n", suiteRunSummary.TestsError)
	fmt.Fprintf(tw, "Suite:\t%s\n", suiteRunSummary.Status)
	fmt.Fprintf(tw, "Total Run Time:\t%.2f sec\n", suiteRunSummary.TotalDuration.Seconds())

	return nil
}
