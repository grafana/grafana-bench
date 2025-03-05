package reporter

import (
	"context"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// SuiteRunReporter defines the methods for reporting a SuiteRunSummary
type SuiteRunReporter interface {
	Report(
		ctx context.Context,
		suiteRun executor.SuiteRun,
		suiteRunSummary executor.SuiteRunSummary,
	) error
}
