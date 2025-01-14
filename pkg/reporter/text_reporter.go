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
	suiteName string,
	suiteRevision string,
	runId string,
	suiteRunId string,
	suiteRun executor.SuiteRunSummary,
) error {
	for _, testRun := range suiteRun.TestRuns {
		fmt.Fprintf(
			r.report,
			"%s/%s ... %s\n",
			testRun.TestFolder,
			testRun.TestFile,
			testRun.Status,
		)
	}
	fmt.Fprintf(r.report, "\nTests executed %d\n", suiteRun.TestsExecuted)
	fmt.Fprintf(r.report, "Tests passed %d\n", suiteRun.TestsPassed)
	fmt.Fprintf(r.report, "Tests failed %d\n", suiteRun.TestsFailed)
	fmt.Fprintf(r.report, "Tests error %d\n", suiteRun.TestsError)
	fmt.Fprintf(r.report, "\nTests suite %s\n", suiteRun.Status)

	return nil
}
