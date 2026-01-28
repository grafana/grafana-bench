package reporter

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/grafana/grafana-bench/pkg/executor"
)

// TextReporter reports a test suite run using structured logs
type TextReporter struct {
	report io.Writer
}

func NewTextReporter(report io.Writer) *TextReporter {
	return &TextReporter{
		report: report,
	}
}

// reportBenchmarks displays benchmark results with metrics in a table format
func (r *TextReporter) reportBenchmarks(tw *tabwriter.Writer, summary executor.SuiteRunSummary) {
	// Create a map of benchmark name -> metrics
	type benchMetrics struct {
		status     executor.TestStatus
		duration   float64
		iterations float64
		nsPerOp    float64
		bytesPerOp float64
		allocsPerOp float64
	}

	benchmarks := make(map[string]*benchMetrics)

	// Initialize from test runs
	for _, testRun := range summary.TestRuns {
		key := testRun.TestFolder + ":" + testRun.TestFile
		benchmarks[key] = &benchMetrics{
			status:   testRun.Status,
			duration: testRun.TotalDuration.Seconds(),
		}
	}

	// Populate metrics
	for _, metric := range summary.Metrics {
		benchName, ok := metric.Labels["benchmark"]
		if !ok {
			continue
		}
		pkg := metric.Labels["package"]
		key := pkg + ":" + benchName

		bench, exists := benchmarks[key]
		if !exists {
			continue
		}

		switch metric.Name {
		case "bench_go_benchmark_iterations":
			bench.iterations = metric.Value
		case "bench_go_benchmark_ns_per_op":
			bench.nsPerOp = metric.Value
		case "bench_go_benchmark_bytes_per_op":
			bench.bytesPerOp = metric.Value
		case "bench_go_benchmark_allocs_per_op":
			bench.allocsPerOp = metric.Value
		}
	}

	// Display header
	fmt.Fprintf(tw, "\n----------------BENCHMARK RESULTS----------------\n")
	fmt.Fprintf(tw, "Benchmark\tStatus\tIterations\tns/op\tB/op\tallocs/op\n")
	fmt.Fprintf(tw, "---------\t------\t----------\t-----\t----\t---------\n")

	// Display each benchmark
	for _, testRun := range summary.TestRuns {
		key := testRun.TestFolder + ":" + testRun.TestFile
		bench := benchmarks[key]

		fmt.Fprintf(tw, "%s\t%s\t%.0f\t%.2f\t%.0f\t%.0f\n",
			testRun.TestFile,
			strings.ToUpper(string(bench.status)),
			bench.iterations,
			bench.nsPerOp,
			bench.bytesPerOp,
			bench.allocsPerOp,
		)
	}
	fmt.Fprintf(tw, "\n")
}

func (r *TextReporter) Report(
	_ context.Context,
	suiteRun executor.SuiteRun,
	suiteRunSummary executor.SuiteRunSummary,
) error {
	tw := tabwriter.NewWriter(r.report, 5, 0, 1, ' ', 0)
	defer tw.Flush()

	// Check if this is a benchmark run by looking for benchmark metrics
	hasBenchmarkMetrics := false
	for _, metric := range suiteRunSummary.Metrics {
		if strings.HasPrefix(metric.Name, "bench_go_benchmark_") {
			hasBenchmarkMetrics = true
			break
		}
	}

	if hasBenchmarkMetrics {
		// Display benchmark results with metrics
		r.reportBenchmarks(tw, suiteRunSummary)
	} else {
		// Display regular test results
		for _, testRun := range suiteRunSummary.TestRuns {
			fmt.Fprintf(
				tw,
				"[%s]\t%.2f sec\t%s:\t%s\n",
				strings.ToUpper(string(testRun.Status)),
				testRun.TotalDuration.Seconds(),
				testRun.TestFolder,
				testRun.TestFile,
			)
		}
	}

	testsByStatus := make(map[executor.TestStatus][]string, 0)
	for _, testRun := range suiteRunSummary.TestRuns {
		if testRun.Status == executor.TestPassed || testRun.Status == executor.TestSkipped {
			continue
		}

		// collect tests that didn't pass
		tests := testsByStatus[testRun.Status]
		tests = append(tests, fmt.Sprintf("%s:\t%s", testRun.TestFolder, testRun.TestFile))
		testsByStatus[testRun.Status] = tests
	}

	statuses := []executor.TestStatus{executor.TestError, executor.TestFailed, executor.TestFlaky}
	for _, status := range statuses {
		tests := testsByStatus[status]
		if len(tests) > 0 {
			fmt.Fprintf(tw, "\n--------------%s TESTS--------------\n", strings.ToUpper(string(status)))
			for _, test := range tests {
				fmt.Fprintf(tw, "%s\n", test)
			}
		}
	}

	fmt.Fprintf(tw, "\n----------------SUMMARY----------------\n")
	fmt.Fprintf(tw, "Executed:\t%d\n", suiteRunSummary.TestsExecuted)
	fmt.Fprintf(tw, "Passed:\t%d\n", suiteRunSummary.TestsPassed)
	fmt.Fprintf(tw, "Flaky:\t%d\n", suiteRunSummary.TestsFlaky)
	fmt.Fprintf(tw, "Failed:\t%d\n", suiteRunSummary.TestsFailed)
	fmt.Fprintf(tw, "Errors:\t%d\n", suiteRunSummary.TestsError)
	fmt.Fprintf(tw, "Suite:\t%s\n", suiteRunSummary.Status)
	fmt.Fprintf(tw, "Total Run Time:\t%.2f sec\n", suiteRunSummary.TotalDuration.Seconds())

	if len(suiteRun.Attributes) > 0 {
		fmt.Fprintf(tw, "\n--------------ATTRIBUTES---------------\n")
		for key, value := range suiteRun.Attributes {
			fmt.Fprintf(tw, "%s:\t%s\n", key, value)
		}
	}

	return nil
}
