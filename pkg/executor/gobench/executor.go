// Package gobench implements a go benchmark test runner
package gobench

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/grafana/grafana-bench/pkg/executor"
	"github.com/grafana/grafana-bench/pkg/metrics"
)

// GoBenchExecutor implements a TestExecutor for go benchmarks
type GoBenchExecutor struct {
	log      *slog.Logger
	opts     GoBenchExecutorOptions
}

func NewGoBenchExecutor(log *slog.Logger, opts GoBenchExecutorOptions) *GoBenchExecutor {
	return &GoBenchExecutor{
		log:  log,
		opts: opts,
	}
}

// Name returns the name of the executor
func (e *GoBenchExecutor) Name() string {
	return "gobench"
}

// ExecTestSuite executes a benchmark suite and reports the results
func (e *GoBenchExecutor) ExecTestSuite(
	ctx context.Context,
	suite executor.TestSuite,
	env map[string]string,
) (executor.SuiteRunSummary, error) {
	workDir := filepath.Join(suite.BaseDir, suite.Path)
	stdOut, err := e.runGoBench(ctx, workDir)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to execute benchmarks: %w", err)
	}

	// Parse benchmark results
	benchmarks, err := ParseJsonOutput(stdOut)
	if err != nil {
		return executor.SuiteRunSummary{}, fmt.Errorf("failed to parse benchmark output: %w", err)
	}

	// Convert benchmarks to suite summary
	summary := e.convertToSuiteRunSummary(benchmarks, suite)

	return summary, nil
}

// runGoBench executes go test with benchmark flags
func (e *GoBenchExecutor) runGoBench(ctx context.Context, workdir string) (io.Reader, error) {
	// Build command: go test -bench [pattern] -json [flags] [packages]
	cmdArgs := []string{"test"}

	// Add benchmark pattern
	benchPattern := e.opts.BenchPattern
	if benchPattern == "" {
		benchPattern = "."
	}
	cmdArgs = append(cmdArgs, "-bench", benchPattern)

	// Add -benchmem if enabled
	if e.opts.BenchMem {
		cmdArgs = append(cmdArgs, "-benchmem")
	}

	// Add -benchtime if specified
	if e.opts.BenchTime != "" {
		cmdArgs = append(cmdArgs, "-benchtime", e.opts.BenchTime)
	}

	// Add -count if specified
	if e.opts.Count > 0 {
		cmdArgs = append(cmdArgs, "-count", strconv.Itoa(e.opts.Count))
	}

	// Add custom go test arguments
	cmdArgs = append(cmdArgs, e.opts.GoArgs...)

	// Add package patterns (default to current directory)
	packages := e.opts.Packages
	if len(packages) == 0 {
		packages = []string{"./..."}
	}
	cmdArgs = append(cmdArgs, packages...)

	// Capture output in json format
	cmdArgs = append(cmdArgs, "-json")

	// Add benchmark-specific arguments via -args
	if len(e.opts.BenchArgs) > 0 {
		cmdArgs = append(cmdArgs, "-args")
		cmdArgs = append(cmdArgs, e.opts.BenchArgs...)
	}

	cmd := exec.CommandContext(ctx, "go", cmdArgs...)
	cmd.Env = os.Environ()
	cmd.Dir = workdir

	stdErr := &bytes.Buffer{}
	stdOut := &bytes.Buffer{}
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr

	e.log.Debug("executing go benchmark", "args", cmd.Args)
	err := cmd.Run()

	// Benchmarks can fail (exit code 1) if tests fail, but we still want to parse results
	if err != nil && cmd.ProcessState.ExitCode() != 1 {
		e.log.Debug("go benchmark failed", "err", err, "stderr", stdErr.String())
		return nil, fmt.Errorf("failed to execute go benchmark command: %w", err)
	}

	return stdOut, nil
}

// convertToSuiteRunSummary converts benchmark results to SuiteRunSummary
func (e *GoBenchExecutor) convertToSuiteRunSummary(
	benchmarks []BenchmarkRunSummary,
	suite executor.TestSuite,
) executor.SuiteRunSummary {
	summary := executor.SuiteRunSummary{
		SuiteName:     suite.Name,
		SuiteRevision: suite.Revision,
		TestRuns:      make([]executor.TestRunSummary, 0, len(benchmarks)),
		Metrics:       make([]metrics.Metric, 0),
	}

	// Track earliest start time and latest end time
	for _, bench := range benchmarks {
		if summary.StartTime.IsZero() || bench.StartTime.Before(summary.StartTime) {
			summary.StartTime = bench.StartTime
		}

		endTime := bench.StartTime.Add(bench.TotalDuration)
		summaryEndTime := summary.StartTime.Add(summary.TotalDuration)
		if endTime.After(summaryEndTime) {
			summary.TotalDuration = endTime.Sub(summary.StartTime)
		}

		// Convert benchmark to TestRunSummary for reporting
		testRun := executor.TestRunSummary{
			TestFolder:       bench.BenchmarkFolder,
			TestFile:         bench.BenchmarkName,
			StartTime:        bench.StartTime,
			Status:           bench.Status,
			ExitMessage:      bench.ExitMessage,
			TotalDuration:    bench.TotalDuration,
			ScenarioDuration: bench.TotalDuration,
		}
		summary.TestRuns = append(summary.TestRuns, testRun)

		// Update counters
		summary.TestsExecuted++
		switch bench.Status {
		case executor.TestPassed:
			summary.TestsPassed++
		case executor.TestFailed:
			summary.TestsFailed++
		case executor.TestError:
			summary.TestsError++
		}

		// Add benchmark-specific metrics
		summary.Metrics = append(summary.Metrics, e.createBenchmarkMetrics(bench)...)
	}

	return summary
}

// createBenchmarkMetrics creates Prometheus metrics for a benchmark result
func (e *GoBenchExecutor) createBenchmarkMetrics(bench BenchmarkRunSummary) []metrics.Metric {
	// Benchmark-specific labels (standard labels like service, service_version, etc.
	// are automatically added by the Prometheus reporter)
	labels := map[string]string{
		"benchmark": bench.BenchmarkName,
		"package":   bench.BenchmarkFolder,
		"procs":     strconv.Itoa(bench.Procs),
	}

	timestamp := bench.StartTime.UnixMilli()

	metricsSlice := []metrics.Metric{
		{
			Name:      "bench_go_benchmark_ns_per_op",
			Value:     bench.NsPerOp,
			Labels:    copyLabels(labels),
			Timestamp: timestamp,
		},
		{
			Name:      "bench_go_benchmark_iterations",
			Value:     float64(bench.Iterations),
			Labels:    copyLabels(labels),
			Timestamp: timestamp,
		},
	}

	// Add memory metrics if available
	if bench.BytesPerOp > 0 {
		metricsSlice = append(metricsSlice, metrics.Metric{
			Name:      "bench_go_benchmark_bytes_per_op",
			Value:     bench.BytesPerOp,
			Labels:    copyLabels(labels),
			Timestamp: timestamp,
		})
	}

	if bench.AllocsPerOp > 0 {
		metricsSlice = append(metricsSlice, metrics.Metric{
			Name:      "bench_go_benchmark_allocs_per_op",
			Value:     float64(bench.AllocsPerOp),
			Labels:    copyLabels(labels),
			Timestamp: timestamp,
		})
	}

	return metricsSlice
}

// copyLabels creates a copy of a label map
func copyLabels(labels map[string]string) map[string]string {
	copied := make(map[string]string, len(labels))
	for k, v := range labels {
		copied[k] = v
	}
	return copied
}
