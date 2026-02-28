package gobench

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
