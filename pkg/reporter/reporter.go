package reporter

import (
	"context"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// SuiteRunReporter defines the methods for reporting a SuiteRunSummary
type SuiteRunReporter interface {
	Report(
		ctx context.Context,
		runId string,
		suiteName string,
		suiteRevision string,
		suiteRunId string,
		suiteRun executor.SuiteRunSummary,
	) error
}