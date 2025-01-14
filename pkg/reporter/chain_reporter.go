package reporter

import (
	"context"
	"errors"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// ChainReporter chains multiple reporters
type ChainReporter struct {
	reporters []SuiteRunReporter
}

func NewChainReporter(reporters ...SuiteRunReporter) SuiteRunReporter {
	return &ChainReporter{
		reporters: reporters,
	}
}

func (c *ChainReporter) Report(
	ctx context.Context,
	suiteName string,
	suiteRevision string,
	runId string,
	suiteRunId string,
	suiteRun executor.SuiteRunSummary,
) error {
	errs := []error{}
	for _, reporter := range c.reporters {
		err := reporter.Report(ctx, suiteName, suiteRevision, runId, suiteRunId, suiteRun)
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}