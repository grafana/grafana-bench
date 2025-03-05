// Package notifier provides methods for sending notifications.
package notifier

import (
	"context"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// Notifier defines the interface for sending notifications
type Notifier interface {
	// Notify sends a notification of test results to a recipient
	Notify(
		ctx context.Context,
		recipient string,
		suiteRunId string,
		testRuns []executor.TestRunSummary,
	) error
}
