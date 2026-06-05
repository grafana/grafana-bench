# Writing Go tests

Bench wraps `go test` and normalizes the output into consistent reports, Prometheus metrics, and Slack notifications — the same as any other test runner.

## Quickstart

1. Write a standard Go test:

```go
// health_test.go
package myservice_test

import (
    "net/http"
    "testing"
)

func TestHealthEndpoint(t *testing.T) {
    resp, err := http.Get("http://localhost:3000/api/health")
    if err != nil {
        t.Fatalf("request failed: %v", err)
    }
    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected 200, got %d", resp.StatusCode)
    }
}
```

2. Run it with bench:

```sh
grafana-bench test \
  --service my-service \
  --service-version v1.0.0 \
  --suite-name my-repo/go-tests \
  --test-runner go \
  --suite-path ./
```

Expected output:

```text
health_test.go/TestHealthEndpoint ... passed

Tests executed 1
Tests passed 1
Tests failed 0

Tests suite passed
```

## How it works

Bench runs `go test -json` on your packages, parses the structured output, and feeds results into its reporters. Your tests don't need any changes — any standard Go test works.

## Selecting packages

By default bench runs tests in the directory specified by `--suite-path`. Use `--go-test-packages` to target specific packages, the same way you would with `go test` directly.

```sh
# Run all packages recursively
grafana-bench test \
  --service my-service \
  --service-version v1.0.0 \
  --suite-name my-repo/go-tests \
  --test-runner go \
  --go-test-packages ./...

# Run a specific package
grafana-bench test \
  --service my-service \
  --service-version v1.0.0 \
  --suite-name my-repo/go-tests \
  --test-runner go \
  --go-test-packages ./pkg/api/...

# Run multiple package patterns
grafana-bench test \
  --service my-service \
  --service-version v1.0.0 \
  --suite-name my-repo/go-tests \
  --test-runner go \
  --go-test-packages ./pkg/api/... \
  --go-test-packages ./pkg/store/...
```

Use `--suite-path` to change the working directory for the `go test` command:

```sh
grafana-bench test \
  --service my-service \
  --service-version v1.0.0 \
  --suite-name my-repo/go-tests \
  --test-runner go \
  --suite-path ./tests \
  --go-test-packages ./...
```

## Passing arguments to go test

Use `--go-args` for flags that go to the `go test` command itself (build tags, race detector, timeout, etc.):

```sh
# Run with build tags
grafana-bench test \
  --service my-service \
  --service-version v1.0.0 \
  --suite-name my-repo/integration-tests \
  --test-runner go \
  --go-test-packages ./... \
  --go-args "-tags=integration"

# Run with race detector and custom timeout
grafana-bench test \
  --service my-service \
  --service-version v1.0.0 \
  --suite-name my-repo/go-tests \
  --test-runner go \
  --go-test-packages ./... \
  --go-args "-race -timeout=10m"

# Combine multiple flags
grafana-bench test \
  --service my-service \
  --service-version v1.0.0 \
  --suite-name my-repo/go-tests \
  --test-runner go \
  --go-test-packages ./... \
  --go-args "-tags=slow -race -timeout=30m"
```

## Passing arguments to your tests

Use `--go-test-args` for flags consumed by the test code itself (passed via `go test -args`):

```sh
grafana-bench test \
  --service my-service \
  --service-version v1.0.0 \
  --suite-name my-repo/integration-tests \
  --test-runner go \
  --go-test-packages ./... \
  --go-test-args "-service-url=http://localhost:3000" \
  --go-test-args "-service-user=admin"
```

Your test reads them with `flag`:

```go
var serviceURL = flag.String("service-url", "http://localhost:3000", "Service URL")

func TestDashboard(t *testing.T) {
    resp, err := http.Get(*serviceURL + "/api/dashboards/home")
    // ...
}
```

## Retries and flaky test detection

Use `--go-retries` to retry failed tests automatically. Tests that fail on the first attempt but pass on a retry are reported as **flaky** rather than failed.

```sh
grafana-bench test \
  --service my-service \
  --service-version v1.0.0 \
  --suite-name my-repo/go-tests \
  --test-runner go \
  --go-test-packages ./... \
  --go-retries 3
```

Output for a flaky test:

```text
pkg/api/api_test.go/TestDashboardLoad ... flaky (passed on retry 2)

Tests executed 5
Tests passed 4
Tests failed 0
Tests flaky 1
```

Flaky results are also exported as the `bench_test_run_flaky` Prometheus metric, letting you track flakiness trends over time.

## GitHub Actions example

For `--service-version`, use the version of the service under test if available. For self-contained unit tests with no external service, use the git SHA:

```yaml
name: Go Tests

on:
  push:
    branches: [main]
  pull_request:

jobs:
  go-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: stable

      - name: Setup Grafana Bench
        uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@8378935874aed527e7a5f4e505d67b61b582a9ea
        with:
          version: 'v1.0.11'

      - name: Run tests
        run: |
          grafana-bench test \
            --service my-service \
            --service-version ${{ github.sha }} \
            --suite-name my-repo/go-tests \
            --test-runner go \
            --go-test-packages ./... \
            --run-stage ci \
            --prometheus-metrics
```

> **Note:** For performance-sensitive integration tests, avoid `ubuntu-latest` (shared, underpowered). Use a dedicated runner like `ubuntu-x64-medium`.

## Configuration file

You can set Go test options in `bench.yaml` to avoid repeating flags:

```yaml
# bench.yaml
suite:
  name: "my-repo/go-tests"
  path: "."

service:
  name: "my-service"
  version: "v1.0.11"

test:
  runner: "go"

go:
  packages:
    - "./..."
  args:
    - "-tags=integration"
    - "-timeout=10m"
  retries: 2
```

> **Note:** Flags on the command line and environment variables take precedence over `bench.yaml` values.

## Flag reference

| Flag | Description |
|---|---|
| `--service` | **Required.** Name of the service being tested |
| `--service-version` | **Required.** Version of the service (use git SHA if no explicit version) |
| `--test-runner go` | Select the Go test runner |
| `--suite-name` | **Required.** Test suite name for logs and metrics |
| `--suite-path` | Working directory for `go test` (default: `.`) |
| `--go-test-packages` | Package patterns to test (repeatable) |
| `--go-args` | Arguments for `go test` (e.g., `-tags=integration -race`) |
| `--go-test-args` | Arguments passed to the test binary via `-args` |
| `--go-retries` | Number of retries for failed tests |

## Related pages

- [Writing Go benchmarks](writing_go_benchmarks.md) - Performance benchmarking with metrics export
- [bench test reference](bench_test.md) - Full flag reference
- [GitHub Actions integration](github_actions.md) - CI setup
- [Metrics guide](metrics.md) - Export results to Prometheus
- [Slack notifications](notifications.md) - Alert your team on failures
