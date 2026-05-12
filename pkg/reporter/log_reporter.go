package reporter

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/utils/format"
)

const (
	TextLog = "text"
	JSONLog = "json"
)

// LogReporter reports a test suite run using structured logs
type LogReporter struct {
	Log *slog.Logger
}

func NewLogReporter(format string, attr []any) (*LogReporter, error) {
	var log *slog.Logger

	switch format {
	case "json":
		log = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	case "text":
		log = slog.New(slog.NewTextHandler(os.Stdout, nil))
	default:
		return nil, fmt.Errorf("unsupported log format: %s", format)
	}

	log = log.With(attr...)

	return &LogReporter{
		Log: log,
	}, nil
}

func (r *LogReporter) Report(
	_ context.Context,
	suiteRun executor.SuiteRun,
	summary executor.SuiteRunSummary,
) error {

	// Build log attributes, conditionally including serviceUrl only for service endpoint tests
	logAttrs := []any{
		"runId", suiteRun.Id,
		"suiteName", summary.SuiteName,
		"suiteRevision", summary.SuiteRevision,
		"runStage", suiteRun.RunStage,
		"testExecutor", suiteRun.TestExecutor,
		"benchRevision", suiteRun.BenchRevision,
	}

	// Only include serviceUrl for service endpoint tests (k6, playwright)
	if suiteRun.ServiceURL != "" {
		logAttrs = append(logAttrs, "serviceUrl", suiteRun.ServiceURL)
	}

	logAttrs = append(logAttrs, "serviceVersion", suiteRun.ServiceVersion)

	log := r.Log.With(logAttrs...)

	for order, testRun := range summary.TestRuns {
		testRunId := fmt.Sprintf("%s-%d", suiteRun.Id, order)

		testRunAttrs := []any{
			"folder", testRun.TestFolder,
			"testFile", testRun.TestFile,
			"iterations", testRun.Iterations,
			"scenarioDuration", format.PrettyMS(testRun.ScenarioDuration),
			"totalDuration", format.PrettyMS(testRun.TotalDuration),
			"status", testRun.Status,
			"attempts", testRun.Attempts,
			"maxAttempts", testRun.MaxAttempts,
			"exitMessage", testRun.ExitMessage,
			"order", strconv.Itoa(order),
		}
		for k, v := range testRun.Attributes {
			testRunAttrs = append(testRunAttrs, k, v)
		}

		log.With(testRunAttrs...).Info("testRun", "testRun", testRunId)
	}

	var anyFailures = (summary.TestsFailed + summary.TestsError) > 0

	suiteRunAttrs := []any{
		"startTime", summary.StartTime.Format(time.RFC3339),
		"totalScenarioDurations", summary.ScenariosDuration,
		"duration", summary.TotalDuration,
		"testsExecuted", summary.TestsExecuted,
		"testsPassed", summary.TestsPassed,
		"testsFlaky", summary.TestsFlaky,
		"testsFailed", summary.TestsFailed,
		"testsError", summary.TestsError,
	}

	for _, v := range summary.Metrics {
		suiteRunAttrs = append(suiteRunAttrs, v.Name, v.Value)
	}

	for k, v := range suiteRun.Attributes {
		suiteRunAttrs = append(suiteRunAttrs, k, v)
	}

	log.With(suiteRunAttrs...).Info("suiteRun", "anyFailures", anyFailures)

	return nil
}
