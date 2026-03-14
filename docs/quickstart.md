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

Pick the runner that matches your tests:

- [Writing K6 tests](writing_k6_api_tests.md) - API testing with JavaScript
- [Writing Playwright tests](writing_pw_tests.md) - End-to-end browser testing
- [Writing Go tests](writing_go_tests.md) - Unit and integration tests
- [Writing Go benchmarks](writing_go_benchmarks.md) - Performance benchmarks

## Add to CI

See the [GitHub Actions integration guide](github_actions.md) to add bench to your workflow.
