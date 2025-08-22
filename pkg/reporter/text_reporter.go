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
			"[%s]\t%s\t%s/%s\n",
			strings.ToUpper(string(testRun.Status)),
			testRun.TotalDuration.String(),
			testRun.TestFolder,
			testRun.TestFile,
		)
	}

	fmt.Fprintf(tw, "\n----------------SUMMARY----------------\n")
	fmt.Fprintf(tw, "Executed:\t%d\n", suiteRunSummary.TestsExecuted)
	fmt.Fprintf(tw, "Passed:\t%d\n", suiteRunSummary.TestsPassed)
	fmt.Fprintf(tw, "Flaky:\t%d\n", suiteRunSummary.TestsFlaky)
	fmt.Fprintf(tw, "Failed:\t%d\n", suiteRunSummary.TestsFailed)
	fmt.Fprintf(tw, "Errors:\t%d\n", suiteRunSummary.TestsError)
	fmt.Fprintf(tw, "Suite:\t%s\n", suiteRunSummary.Status)

	return nil
}
