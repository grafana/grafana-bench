package gobench

import (
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// BenchmarkRunSummary represents the results of a single benchmark run
type BenchmarkRunSummary struct {
	// BenchmarkFolder is the package path where the benchmark is located
	BenchmarkFolder string
	// BenchmarkName is the name of the benchmark function
	BenchmarkName string
	// StartTime is when the benchmark execution started
	StartTime time.Time
	// Status indicates whether the benchmark passed, failed, or errored
	Status executor.TestStatus
	// ExitMessage contains error or failure messages
	ExitMessage string
	// Iterations is the number of benchmark iterations (N)
	Iterations int64
	// NsPerOp is the nanoseconds per operation
	NsPerOp float64
	// BytesPerOp is the bytes allocated per operation (requires -benchmem)
	BytesPerOp float64
	// AllocsPerOp is the number of allocations per operation (requires -benchmem)
	AllocsPerOp int64
	// MBPerSec is the throughput in MB/s (if reported)
	MBPerSec float64
	// TotalDuration is the total time taken for the benchmark
	TotalDuration time.Duration
	// Procs is the GOMAXPROCS value (from -N suffix)
	Procs int
}

// GoBenchExecutorOptions contains configuration options for the Go benchmark executor
type GoBenchExecutorOptions struct {
	// Packages is the list of package patterns to benchmark (e.g., "./...")
	Packages []string
	// GoArgs are additional arguments passed to 'go test' (e.g., "-tags", "-race")
	GoArgs []string
	// BenchArgs are arguments passed to benchmarks via -args
	BenchArgs []string
	// BenchPattern is the regex pattern for benchmark names (default ".")
	BenchPattern string
	// BenchTime is the -benchtime value (e.g., "10s", "100x")
	BenchTime string
	// BenchMem enables -benchmem to capture memory statistics
	BenchMem bool
	// Count is the number of times to run each benchmark
	Count int
}
