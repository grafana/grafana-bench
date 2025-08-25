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
			"[%s]\t%.2f sec\t%s/%s\n",
			strings.ToUpper(string(testRun.Status)),
			testRun.TotalDuration.Seconds(),
			testRun.TestFolder,
			testRun.TestFile,
		)
	}

	unsuccessfulTests := make(map[executor.TestStatus][]string, 0)
	for _, testRun := range suiteRunSummary.TestRuns {
		if testRun.Status == executor.TestPassed {
			continue
		}

		if unsuccessfulTests[testRun.Status] == nil {
			unsuccessfulTests[testRun.Status] = make([]string, 0)
		}

		unsuccessfulTests[testRun.Status] = append(
			unsuccessfulTests[testRun.Status],
			fmt.Sprintf("%s:\t%s", testRun.TestFolder, testRun.TestFile),
		)
	}

	if len(unsuccessfulTests) > 0 {
		if flakyTests, ok := unsuccessfulTests[executor.TestFlaky]; ok {
			fmt.Fprintf(tw, "\n--------------FLAKY TESTS--------------\n")
			for _, flakyTest := range flakyTests {
				fmt.Fprintf(tw, "%s\n", flakyTest)
			}
		}

		if failedTests, ok := unsuccessfulTests[executor.TestFailed]; ok {
			fmt.Fprintf(tw, "\n--------------FAILED TESTS-------------\n")
			for _, failedTest := range failedTests {
				fmt.Fprintf(tw, "%s\n", failedTest)
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
