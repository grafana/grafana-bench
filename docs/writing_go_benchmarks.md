# Writing Go Benchmarks for Grafana Bench

This guide covers how to write Go benchmarks that integrate with Grafana Bench for performance tracking and regression detection.

## Overview

Grafana Bench supports running Go benchmarks via the `gobench` test runner, which executes `go test -bench` and exports performance metrics (ns/op, B/op, allocs/op) to Prometheus, text reports, and JSON logs.

## CI/CD Integration

**NOTE:** when running in ci, do NOT use the default ubuntu-latest github runner. This is a shared runner that is very
small. Use a self-hosted runner like ubuntu-x64-medium

Example GitHub Actions workflow:

```yaml
name: Performance Benchmarks

on:
  push:
    branches: [main]
  pull_request:

jobs:
  benchmark:
    # USE A DEDICATED RUNNER
    # do _not_ use ubuntu-latest
    runs-on: ubuntu-x64-medium
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: 'stable'

      - name: Setup bench + Prometheus secrets
        uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@057477c3d586996c1fc3f38772760c34a68d2859
        with:
          version: latest

      - name: Run benchmarks
        run: |
          grafana-bench test \
            --service myservice \
            --service-version ci \
            --test-runner gobench \
            --suite-name myservice/benchmarks \
            --suite-path ./benchmarks \
            --gobench-time 10s \
            --prometheus-metrics \
            --run-stage ci
```

## Basic Benchmark Structure

Go benchmarks follow standard Go testing conventions:

```go
package mypackage

import "testing"

func BenchmarkMyFunction(b *testing.B) {
    // Setup code (not measured)
    data := prepareTestData()

    // Reset timer after setup
    b.ResetTimer()

    // Benchmark loop (measured)
    for i := 0; i < b.N; i++ {
        MyFunction(data)
    }
}
```

## Running Benchmarks with Grafana Bench

### Basic Usage

```bash
grafana-bench test \
  --service myservice \
  --service-version v1.0.0 \
  --test-runner gobench \
  --suite-name myservice/benchmarks \
  --suite-path ./benchmarks \
  --run-stage ci
```

### With Prometheus Metrics

```bash
grafana-bench test \
  --service myservice \
  --service-version v1.0.0 \
  --test-runner gobench \
  --suite-name myservice/api-benchmarks \
  --suite-path ./benchmarks \
  --gobench-pattern "BenchmarkAPI" \
  --gobench-time 10s \
  --gobench-count 3 \
  --prometheus-metrics \
  --run-stage ci
```

## Benchmark Flags

- `--gobench-packages` - Package patterns (e.g., `./...`, `./pkg/...`)
- `--gobench-pattern` - Benchmark name pattern (regex, default `.`)
- `--gobench-time` - Benchmark duration (e.g., `10s`, `100x`)
- `--gobench-mem` - Enable memory statistics (default `true`)
- `--gobench-count` - Number of runs per benchmark
- `--gobench-args` - Additional `go test` arguments
- `--gobench-bench-args` - Arguments passed to benchmarks via `-args`

## Best Practices

### 1. Use Sub-benchmarks for Variations

```go
func BenchmarkJSONMarshal(b *testing.B) {
    sizes := []int{100, 1000, 10000}

    for _, size := range sizes {
        b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
            data := generateData(size)
            b.ResetTimer()

            for i := 0; i < b.N; i++ {
                json.Marshal(data)
            }
        })
    }
}
```

### 2. Reset Timer After Setup

Always reset the timer after expensive setup to measure only the relevant code:

```go
func BenchmarkDatabaseQuery(b *testing.B) {
    db := setupDatabase() // Not measured
    query := prepareQuery() // Not measured

    b.ResetTimer() // Start measuring here

    for i := 0; i < b.N; i++ {
        db.Query(query)
    }
}
```

### 3. Prevent Compiler Optimizations

Store results in package-level variables to prevent the compiler from optimizing away your benchmark:

```go
var result int

func BenchmarkExpensiveCalculation(b *testing.B) {
    var r int
    for i := 0; i < b.N; i++ {
        r = expensiveCalculation()
    }
    result = r // Prevent optimization
}
```

### 4. Use ReportAllocs for Memory Tracking

```go
func BenchmarkStringConcat(b *testing.B) {
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _ = "hello" + "world"
    }
}
```

Note: When using `--gobench-mem=true` (default), memory statistics are automatically enabled.

### 5. Benchmark Parallel Operations

For concurrent code, use `RunParallel`:

```go
func BenchmarkConcurrentMap(b *testing.B) {
    m := sync.Map{}

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            m.Store("key", "value")
            m.Load("key")
        }
    })
}
```

## Metrics Exported to Prometheus

Grafana Bench exports the following metrics for each benchmark:

- `bench_go_benchmark_ns_per_op` - Nanoseconds per operation
- `bench_go_benchmark_iterations` - Number of iterations (N)
- `bench_go_benchmark_bytes_per_op` - Bytes allocated per operation (with --gobench-mem)
- `bench_go_benchmark_allocs_per_op` - Allocations per operation (with --gobench-mem)

Each metric includes these labels:

- `benchmark` - Benchmark function name
- `package` - Package path
- `procs` - GOMAXPROCS value
- `service` - Service name (from --service flag)
- `service_version` - Service version (from --service-version flag)
- `suite_name` - Test suite name
- `run_stage` - Execution stage (ci, local, etc.)
- `status` - Benchmark status (passed, failed)

## Example Benchmarks

### API Endpoint Performance

```go
func BenchmarkAPIEndpoints(b *testing.B) {
    endpoints := []string{"/api/users", "/api/products", "/api/orders"}

    for _, endpoint := range endpoints {
        b.Run(endpoint, func(b *testing.B) {
            client := setupHTTPClient()
            b.ResetTimer()

            for i := 0; i < b.N; i++ {
                resp, _ := client.Get("http://localhost:3000" + endpoint)
                resp.Body.Close()
            }
        })
    }
}
```

### Data Structure Performance

```go
func BenchmarkDataStructures(b *testing.B) {
    b.Run("slice_append", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            var s []int
            for j := 0; j < 1000; j++ {
                s = append(s, j)
            }
        }
    })

    b.Run("slice_preallocated", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            s := make([]int, 0, 1000)
            for j := 0; j < 1000; j++ {
                s = append(s, j)
            }
        }
    })
}
```

### Algorithm Complexity

```go
func BenchmarkSearch(b *testing.B) {
    sizes := []int{100, 1000, 10000}

    for _, size := range sizes {
        data := generateSortedSlice(size)
        target := data[size/2]

        b.Run(fmt.Sprintf("linear_%d", size), func(b *testing.B) {
            for i := 0; i < b.N; i++ {
                linearSearch(data, target)
            }
        })

        b.Run(fmt.Sprintf("binary_%d", size), func(b *testing.B) {
            for i := 0; i < b.N; i++ {
                binarySearch(data, target)
            }
        })
    }
}
```

## Troubleshooting

### Benchmarks Not Found

Ensure your benchmark functions:

- Start with `Benchmark` prefix
- Accept `*testing.B` parameter
- Are in `*_test.go` files

### Inconsistent Results

- Increase `--gobench-time` for more stable results
- Use `--gobench-count` to run multiple iterations
- Ensure benchmarks are deterministic
- Disable CPU frequency scaling on benchmark machines

### Memory Statistics Not Showing

Verify `--gobench-mem=true` is set (default) and your benchmarks call `b.ReportAllocs()` or memory operations are significant enough to measure.

## References

- [Go Testing Package](https://pkg.go.dev/testing)
- [Go Benchmark Guide](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
- [Grafana Bench Metrics Documentation](metrics.md)
