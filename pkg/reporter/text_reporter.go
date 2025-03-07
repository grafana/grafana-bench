package reporter

import (
	"context"
	"fmt"
	"io"

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
	for _, testRun := range suiteRunSummary.TestRuns {
		fmt.Fprintf(
			r.report,
			"%s/%s ... %s\n",
			testRun.TestFolder,
			testRun.TestFile,
			testRun.Status,
		)
	}
	fmt.Fprintf(r.report, "\nTests executed %d\n", suiteRunSummary.TestsExecuted)
	fmt.Fprintf(r.report, "Tests passed %d\n", suiteRunSummary.TestsPassed)
	fmt.Fprintf(r.report, "Tests flaky %d\n", suiteRunSummary.TestsFlaky)
	fmt.Fprintf(r.report, "Tests failed %d\n", suiteRunSummary.TestsFailed)
	fmt.Fprintf(r.report, "Tests error %d\n", suiteRunSummary.TestsError)
	fmt.Fprintf(r.report, "\nTests suite %s\n", suiteRunSummary.Status)

	return nil
}
