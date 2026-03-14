package gobench

import (
	"log/slog"
	"os"
	"testing"

	"github.com/grafana/grafana-bench/pkg/executor"
	gobenchparser "github.com/grafana/grafana-bench/pkg/parser/gobench"
)

func TestNewGoBenchExecutor(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	opts := GoBenchExecutorOptions{
		Packages:     []string{"./..."},
		BenchPattern: ".",
		BenchMem:     true,
		Count:        1,
	}

	exec := NewGoBenchExecutor(log, opts)

	if exec == nil {
		t.Fatal("expected non-nil executor")
	}

	if exec.Name() != "gobench" {
		t.Errorf("expected name 'gobench', got %q", exec.Name())
	}
}

func TestCreateBenchmarkMetrics(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	exec := NewGoBenchExecutor(log, GoBenchExecutorOptions{})

	bench := gobenchparser.BenchmarkRunSummary{
		BenchmarkFolder: "github.com/example/pkg",
		BenchmarkName:   "BenchmarkTest",
		Status:          executor.TestPassed,
		Iterations:      1000000,
		NsPerOp:         1234.5,
		BytesPerOp:      456,
		AllocsPerOp:     7,
		Procs:           8,
	}

	metrics := exec.createBenchmarkMetrics(bench)

	// Should have at least 4 metrics (ns/op, iterations, bytes/op, allocs/op)
	if len(metrics) < 4 {
		t.Errorf("expected at least 4 metrics, got %d", len(metrics))
	}

	// Verify metric names and values
	metricMap := make(map[string]float64)
	for _, m := range metrics {
		metricMap[m.Name] = m.Value

		// Verify labels are present
		if m.Labels["benchmark"] != "BenchmarkTest" {
			t.Errorf("expected benchmark label 'BenchmarkTest', got %q", m.Labels["benchmark"])
		}
		// Only benchmark label should be present
		if len(m.Labels) != 1 {
			t.Errorf("expected only benchmark label, got %d labels: %v", len(m.Labels), m.Labels)
		}
	}

	// Verify metric values
	if metricMap["go_benchmark_ns_per_op"] != 1234.5 {
		t.Errorf("expected ns_per_op 1234.5, got %f", metricMap["go_benchmark_ns_per_op"])
	}

	if metricMap["go_benchmark_iterations"] != 1000000 {
		t.Errorf("expected iterations 1000000, got %f", metricMap["go_benchmark_iterations"])
	}

	if metricMap["go_benchmark_bytes_per_op"] != 456 {
		t.Errorf("expected bytes_per_op 456, got %f", metricMap["go_benchmark_bytes_per_op"])
	}

	if metricMap["go_benchmark_allocs_per_op"] != 7 {
		t.Errorf("expected allocs_per_op 7, got %f", metricMap["go_benchmark_allocs_per_op"])
	}
}

func TestConvertToSuiteRunSummary(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	exec := NewGoBenchExecutor(log, GoBenchExecutorOptions{})

	benchmarks := []gobenchparser.BenchmarkRunSummary{
		{
			BenchmarkFolder: "pkg1",
			BenchmarkName:   "BenchmarkA",
			Status:          executor.TestPassed,
			Iterations:      1000,
			NsPerOp:         100,
			Procs:           8,
		},
		{
			BenchmarkFolder: "pkg2",
			BenchmarkName:   "BenchmarkB",
			Status:          executor.TestFailed,
			Iterations:      500,
			NsPerOp:         200,
			Procs:           8,
		},
	}

	suite := executor.TestSuite{
		Name:     "test-suite",
		Revision: "abc123",
	}

	summary := exec.convertToSuiteRunSummary(benchmarks, suite)

	if summary.SuiteName != "test-suite" {
		t.Errorf("expected suite name 'test-suite', got %q", summary.SuiteName)
	}

	if summary.SuiteRevision != "abc123" {
		t.Errorf("expected suite revision 'abc123', got %q", summary.SuiteRevision)
	}

	if summary.TestsExecuted != 2 {
		t.Errorf("expected 2 tests executed, got %d", summary.TestsExecuted)
	}

	if summary.TestsPassed != 1 {
		t.Errorf("expected 1 test passed, got %d", summary.TestsPassed)
	}

	if summary.TestsFailed != 1 {
		t.Errorf("expected 1 test failed, got %d", summary.TestsFailed)
	}

	// Should have metrics for both benchmarks (4 metrics each)
	if len(summary.Metrics) < 4 {
		t.Errorf("expected at least 4 metrics, got %d", len(summary.Metrics))
	}
}
