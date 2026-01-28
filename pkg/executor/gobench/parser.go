package gobench

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/pkg/executor"
)

var (
	ErrInvalidFormat = errors.New("invalid format")

	// benchResultRegex parses benchmark result lines like:
	// BenchmarkName-8   1000000   1234 ns/op   456 B/op   7 allocs/op   89.1 MB/s
	benchResultRegex = regexp.MustCompile(
		`^(Benchmark\S+)-(\d+)\s+` + // BenchmarkName-PROCS
			`(\d+)\s+` + // iterations
			`([\d.]+)\s+ns/op` + // ns/op
			`(?:\s+([\d.]+)\s+B/op)?` + // optional bytes/op
			`(?:\s+(\d+)\s+allocs/op)?` + // optional allocs/op
			`(?:\s+([\d.]+)\s+MB/s)?`, // optional throughput
	)
)

type line struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float32 `json:"Elapsed"`
}

type benchmarkKey struct {
	pkg       string
	benchmark string
}

type benchmarkRuns map[benchmarkKey]*BenchmarkRunSummary

// ParseJsonOutput parses benchmark output from go test -bench -json
func ParseJsonOutput(report io.Reader) ([]BenchmarkRunSummary, error) {
	benchmarks, err := parseBenchmarkRuns(report)
	if err != nil {
		return nil, err
	}

	// Convert map to slice
	results := make([]BenchmarkRunSummary, 0, len(benchmarks))
	for _, bench := range benchmarks {
		results = append(results, *bench)
	}

	return results, nil
}

func parseBenchmarkRuns(report io.Reader) (benchmarkRuns, error) {
	benchmarks := benchmarkRuns{}
	var globalStartTime time.Time
	firstEvent := true

	decoder := json.NewDecoder(report)
	for {
		var line line
		err := decoder.Decode(&line)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, err
		}

		// Track the earliest time seen
		if firstEvent && line.Time != "" {
			t, err := time.Parse(time.RFC3339, line.Time)
			if err == nil {
				globalStartTime = t
				firstEvent = false
			}
		}

		// Process benchmark result lines from output
		if line.Action == "output" && line.Test == "" {
			parseBenchmarkOutputLine(line.Output, line.Package, globalStartTime, benchmarks)
		}

		// Handle failures at the package level (no specific test name)
		if line.Action == "fail" && line.Test == "" && line.Package != "" {
			// Mark all benchmarks in this package as failed if they don't have a status yet
			for key, bench := range benchmarks {
				if key.pkg == line.Package && bench.Status == "" {
					bench.Status = executor.TestError
					bench.ExitMessage = fmt.Sprintf("Package failed: %s", line.Package)
				}
			}
		}

		// Handle individual test/benchmark failures
		if line.Action == "fail" && line.Test != "" {
			key := benchmarkKey{pkg: line.Package, benchmark: line.Test}
			if bench, ok := benchmarks[key]; ok {
				bench.Status = executor.TestFailed
				duration := time.Duration(line.Elapsed * float32(time.Second))
				bench.TotalDuration = duration
			}
		}

		// Handle passes
		if line.Action == "pass" && line.Test != "" {
			key := benchmarkKey{pkg: line.Package, benchmark: line.Test}
			if bench, ok := benchmarks[key]; ok {
				if bench.Status == "" {
					bench.Status = executor.TestPassed
				}
				duration := time.Duration(line.Elapsed * float32(time.Second))
				bench.TotalDuration = duration
			}
		}
	}

	// Mark any benchmarks without a status as passed if they have results
	for _, bench := range benchmarks {
		if bench.Status == "" {
			if bench.Iterations > 0 {
				bench.Status = executor.TestPassed
			} else {
				bench.Status = executor.TestError
				bench.ExitMessage = "No benchmark results captured"
			}
		}
	}

	return benchmarks, nil
}

func parseBenchmarkOutputLine(output, pkg string, startTime time.Time, benchmarks benchmarkRuns) {
	// Trim whitespace and check if it's a benchmark result line
	output = strings.TrimSpace(output)
	if !strings.HasPrefix(output, "Benchmark") {
		return
	}

	matches := benchResultRegex.FindStringSubmatch(output)
	if matches == nil {
		return
	}

	// Extract benchmark name and GOMAXPROCS
	benchmarkName := matches[1]
	procs, _ := strconv.Atoi(matches[2])

	// Extract required metrics
	iterations, _ := strconv.ParseInt(matches[3], 10, 64)
	nsPerOp, _ := strconv.ParseFloat(matches[4], 64)

	// Extract optional memory metrics
	var bytesPerOp float64
	var allocsPerOp int64
	if matches[5] != "" {
		bytesPerOp, _ = strconv.ParseFloat(matches[5], 64)
	}
	if matches[6] != "" {
		allocsPerOp, _ = strconv.ParseInt(matches[6], 10, 64)
	}

	// Extract optional throughput
	var mbPerSec float64
	if matches[7] != "" {
		mbPerSec, _ = strconv.ParseFloat(matches[7], 64)
	}

	key := benchmarkKey{pkg: pkg, benchmark: benchmarkName}
	benchmarks[key] = &BenchmarkRunSummary{
		BenchmarkFolder: pkg,
		BenchmarkName:   benchmarkName,
		StartTime:       startTime,
		Iterations:      iterations,
		NsPerOp:         nsPerOp,
		BytesPerOp:      bytesPerOp,
		AllocsPerOp:     allocsPerOp,
		MBPerSec:        mbPerSec,
		Procs:           procs,
	}
}
