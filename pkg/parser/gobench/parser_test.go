package gobench

import (
	"os"
	"testing"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
)

var testTime = time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

func TestParseJsonOutput(t *testing.T) {
	// Read test fixture
	file, err := os.Open("testdata/benchmark_output.json")
	if err != nil {
		t.Fatalf("failed to open test fixture: %v", err)
	}
	defer file.Close()

	// Parse the output
	benchmarks, err := ParseJsonOutput(file)
	if err != nil {
		t.Fatalf("ParseJsonOutput failed: %v", err)
	}

	// Verify we got the expected benchmarks
	if len(benchmarks) != 3 {
		t.Errorf("expected 3 benchmarks, got %d", len(benchmarks))
	}

	// Create a map for easy lookup
	benchMap := make(map[string]BenchmarkRunSummary)
	for _, bench := range benchmarks {
		benchMap[bench.BenchmarkName] = bench
	}

	// Test BenchmarkStringConcat (with memory stats)
	t.Run("BenchmarkStringConcat", func(t *testing.T) {
		bench, ok := benchMap["BenchmarkStringConcat"]
		if !ok {
			t.Fatal("BenchmarkStringConcat not found")
		}

		if bench.BenchmarkFolder != "github.com/example/pkg" {
			t.Errorf("expected package 'github.com/example/pkg', got %q", bench.BenchmarkFolder)
		}

		if bench.Iterations != 1000000 {
			t.Errorf("expected 1000000 iterations, got %d", bench.Iterations)
		}

		if bench.NsPerOp != 1234 {
			t.Errorf("expected 1234 ns/op, got %f", bench.NsPerOp)
		}

		if bench.BytesPerOp != 456 {
			t.Errorf("expected 456 B/op, got %f", bench.BytesPerOp)
		}

		if bench.AllocsPerOp != 7 {
			t.Errorf("expected 7 allocs/op, got %d", bench.AllocsPerOp)
		}

		if bench.Procs != 8 {
			t.Errorf("expected 8 procs, got %d", bench.Procs)
		}

		if bench.Status != executor.TestPassed {
			t.Errorf("expected status TestPassed, got %v", bench.Status)
		}
	})

	// Test BenchmarkStringBuilder (with memory stats)
	t.Run("BenchmarkStringBuilder", func(t *testing.T) {
		bench, ok := benchMap["BenchmarkStringBuilder"]
		if !ok {
			t.Fatal("BenchmarkStringBuilder not found")
		}

		if bench.Iterations != 5000000 {
			t.Errorf("expected 5000000 iterations, got %d", bench.Iterations)
		}

		if bench.NsPerOp != 234 {
			t.Errorf("expected 234 ns/op, got %f", bench.NsPerOp)
		}

		if bench.BytesPerOp != 64 {
			t.Errorf("expected 64 B/op, got %f", bench.BytesPerOp)
		}

		if bench.AllocsPerOp != 2 {
			t.Errorf("expected 2 allocs/op, got %d", bench.AllocsPerOp)
		}
	})

	// Test BenchmarkJSONMarshal (without memory stats)
	t.Run("BenchmarkJSONMarshal", func(t *testing.T) {
		bench, ok := benchMap["BenchmarkJSONMarshal"]
		if !ok {
			t.Fatal("BenchmarkJSONMarshal not found")
		}

		if bench.Iterations != 500000 {
			t.Errorf("expected 500000 iterations, got %d", bench.Iterations)
		}

		if bench.NsPerOp != 3456 {
			t.Errorf("expected 3456 ns/op, got %f", bench.NsPerOp)
		}

		// Memory stats should be 0 when not present
		if bench.BytesPerOp != 0 {
			t.Errorf("expected 0 B/op, got %f", bench.BytesPerOp)
		}

		if bench.AllocsPerOp != 0 {
			t.Errorf("expected 0 allocs/op, got %d", bench.AllocsPerOp)
		}
	})
}

func TestParseBenchmarkOutputLine(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		expectedName   string
		expectedProcs  int
		expectedIters  int64
		expectedNsOp   float64
		expectedBytesOp float64
		expectedAllocsOp int64
		shouldMatch    bool
	}{
		{
			name:           "full benchmark with memory",
			output:         "BenchmarkTest-8   \t 1000000\t      1234 ns/op\t     456 B/op\t       7 allocs/op\n",
			expectedName:   "BenchmarkTest",
			expectedProcs:  8,
			expectedIters:  1000000,
			expectedNsOp:   1234,
			expectedBytesOp: 456,
			expectedAllocsOp: 7,
			shouldMatch:    true,
		},
		{
			name:           "benchmark without memory",
			output:         "BenchmarkTest-4   \t 500000\t      2345 ns/op\n",
			expectedName:   "BenchmarkTest",
			expectedProcs:  4,
			expectedIters:  500000,
			expectedNsOp:   2345,
			expectedBytesOp: 0,
			expectedAllocsOp: 0,
			shouldMatch:    true,
		},
		{
			name:        "non-benchmark line",
			output:      "=== RUN   BenchmarkTest\n",
			shouldMatch: false,
		},
		{
			name:        "package line",
			output:      "pkg: github.com/example/pkg\n",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			benchmarks := make(benchmarkRuns)
			parseBenchmarkOutputLine(tt.output, "test/pkg", testTime, benchmarks)

			if tt.shouldMatch {
				if len(benchmarks) != 1 {
					t.Fatalf("expected 1 benchmark to be parsed, got %d", len(benchmarks))
				}

				// Get the benchmark result
				var bench *BenchmarkRunSummary
				for _, b := range benchmarks {
					bench = b
					break
				}

				if bench.BenchmarkName != tt.expectedName {
					t.Errorf("expected name %q, got %q", tt.expectedName, bench.BenchmarkName)
				}

				if bench.Procs != tt.expectedProcs {
					t.Errorf("expected procs %d, got %d", tt.expectedProcs, bench.Procs)
				}

				if bench.Iterations != tt.expectedIters {
					t.Errorf("expected iterations %d, got %d", tt.expectedIters, bench.Iterations)
				}

				if bench.NsPerOp != tt.expectedNsOp {
					t.Errorf("expected ns/op %f, got %f", tt.expectedNsOp, bench.NsPerOp)
				}

				if bench.BytesPerOp != tt.expectedBytesOp {
					t.Errorf("expected B/op %f, got %f", tt.expectedBytesOp, bench.BytesPerOp)
				}

				if bench.AllocsPerOp != tt.expectedAllocsOp {
					t.Errorf("expected allocs/op %d, got %d", tt.expectedAllocsOp, bench.AllocsPerOp)
				}
			} else {
				if len(benchmarks) != 0 {
					t.Errorf("expected no benchmarks to be parsed, got %d", len(benchmarks))
				}
			}
		})
	}
}
