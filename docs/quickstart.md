# Quickstart

Get bench installed and running your first test.

## Install

**Binary** (requires Go):

```sh
go install github.com/grafana/grafana-bench@v1.0.3
```

**Docker** — base image (K6, Go tests, Go benchmarks):

```sh
docker pull ghcr.io/grafana/grafana-bench:v1.0.3
```

**Docker** — Playwright image (includes Chromium for browser tests):

```sh
docker pull ghcr.io/grafana/grafana-bench-playwright:v1.0.3
```

## Run your first test

Write a standard Go test:

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

Run it with bench:

```sh
grafana-bench test \
  --service my-service \
  --service-version 1.0.0 \
  --test-runner go \
  --suite-name my-repo/go-tests \
  --suite-path ./
```

## Other runners

Pick the runner that matches your tests:

- [Writing K6 tests](writing_k6_api_tests.md) - API testing with JavaScript
- [Writing Playwright tests](writing_pw_tests.md) - End-to-end browser testing
- [Writing Go tests](writing_go_tests.md) - Unit and integration tests
- [Writing Go benchmarks](writing_go_benchmarks.md) - Performance benchmarks

## Add to CI

See the [GitHub Actions integration guide](github_actions.md) to add bench to your workflow.
