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
	suiteRun executor.SuiteRun,
	summary executor.SuiteRunSummary,
) error {
	errs := []error{}
	for _, reporter := range c.reporters {
		err := reporter.Report(ctx, suiteRun, summary)
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}