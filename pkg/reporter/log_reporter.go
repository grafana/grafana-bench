package reporter

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// LogReporter reports a test suite run using structured logs
type LogReporter struct {
	Log *slog.Logger
}

func NewLogReporter(attr []any) *LogReporter {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	log = log.With(attr...)
	return &LogReporter{
		Log: log,
	}
}
 
func (r *LogReporter) Report(
	_ context.Context,
	runId string,
	suiteRunId string,
	suite executor.TestSuite,
	suiteRun executor.SuiteRunSummary,
) error {
	log := r.Log.With("runId", runId, "suiteRun", suiteRunId)

	for order, testRun := range suiteRun.TestRuns {
		testRunId := fmt.Sprintf("%s-%d", runId, order)
		log.With(suiteLogAttrs(suite)...).
			// TODO: deprecate order attribute
			With("order", strconv.Itoa(order)).
			With(testRunLogAttrs(testRun)...).
			Info("testRun", "testRun", testRunId)
	}

	var anyFailures = (suiteRun.TestsFailed + suiteRun.TestsError) > 0

	log.With(suiteLogAttrs(suite)...).
		With(suiteRunLogAttrs(suiteRun)...).
		Info("suiteRun", "anyFailures", anyFailures)

	return nil
}

// suiteLogAttrs formats suite's attributes as log attributes
func suiteLogAttrs(suite executor.TestSuite) []any {
	return []any{
		"suiteName", suite.Name,
		"suiteRevision", suite.Revision,
	}
}

// suiteRunLogAttrs formats suite run's attributes as log attributes
func suiteRunLogAttrs(suiteRun executor.SuiteRunSummary) []any {
	return []any{
		"startTime", suiteRun.StartTime.Format(time.RFC3339),
		"totalScenarioDurations", suiteRun.ScenariosDuration,
		"duration", suiteRun.TotalDuration,
		"testsExecuted", suiteRun.TestsExecuted,
		"testsPassed", suiteRun.TestsPassed,
		"testsFailed", suiteRun.TestsFailed,
		"testsError", suiteRun.TestsError,
	}
}

// testRunLogAttrs returns the k6RunSummary attributes formatted as log attributes
func testRunLogAttrs(testRun executor.TestRun) []any {
	attrs := []any{
		"folder", testRun.TestFolder,
		"testFile", testRun.TestFile,
		"iterations", testRun.Iterations,
		"setupDuration", prettyMS(testRun.Durations.SetupDuration),
		"scenarioDuration", prettyMS(testRun.Durations.ScenarioDuration),
		"teardownDuration", prettyMS(testRun.Durations.TeardownDuration),
		"totalDuration", prettyMS(testRun.Durations.TotalDuration),
		"status", testRun.Status,
		"exitMessage", testRun.ExitMessage,
	}

	for k, v := range testRun.Attributes {
		attrs = append(attrs, k, v)
	}
	return attrs
}

// prettyMS adds ms suffix to ms float
func prettyMS(ms float32) string {
	duration := time.Duration(ms) * time.Millisecond
	return fmt.Sprintf("%dms", duration.Milliseconds())
}
