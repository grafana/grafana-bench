# Grafana Bench

**Bench** is a CLI tool for standardized test execution and observability. It wraps your
testing tools (K6, Playwright, Go tests, Go benchmarks) and produces consistent structured
logs and Prometheus metrics regardless of what you're testing.

## What it does

Bench normalizes test output into structured logs and Prometheus metrics. Every test runner has a driver (executes tests) and a parser (normalizes output) — you can let bench handle both, or run tests yourself and pipe the output to bench with a built-in or custom parser:

**Structured log** (every test run):

```text
level=info msg="suite complete" suite_name=my-api/smoke service=my-api service_version=1.2.0 run_stage=ci tests_executed=12 tests_passed=11 tests_failed=1 duration_seconds=4.2
```

**Prometheus metrics** (when `--prometheus-metrics` is set):

```text
bench_tests_executed{service="my-api",service_version="1.2.0",suite_name="my-api/smoke",run_stage="ci"} 12
bench_tests_passed{service="my-api",service_version="1.2.0",suite_name="my-api/smoke",run_stage="ci"} 11
bench_tests_failed{service="my-api",service_version="1.2.0",suite_name="my-api/smoke",run_stage="ci"} 1
bench_total_duration_seconds{service="my-api",service_version="1.2.0",suite_name="my-api/smoke",run_stage="ci"} 4.2
```

## Quick start

```sh
# Install
go install github.com/grafana/grafana-bench@latest

# Run K6 tests
grafana-bench test \
  --service my-api \
  --service-version 1.0.0 \
  --service-url http://localhost:3000 \
  --test-runner k6 \
  --suite-name my-api/smoke \
  --suite-path ./tests
```

## Docker images

```sh
# Base image (K6, Go tests, Go benchmarks)
docker pull ghcr.io/grafana/grafana-bench:latest

# Playwright image (includes Chromium for browser tests)
docker pull ghcr.io/grafana/grafana-bench-playwright:latest
```

For versioned pulls (recommended in CI):

```sh
docker pull ghcr.io/grafana/grafana-bench:v1.0.3
docker pull ghcr.io/grafana/grafana-bench-playwright:v1.0.3
```

## Running with Docker

```sh
docker run --rm -e SUITE_REPO_TOKEN --network=host \
  ghcr.io/grafana/grafana-bench-playwright:v1.0.3 test \
  --service my-api \
  --service-url http://localhost:3000 \
  --service-version 1.0.0 \
  --test-runner playwright \
  --suite-name my-repo/e2e \
  --suite-path ./tests
```

For local development with volume mounts (Linux/macOS):

```sh
docker run --rm --network=host --volume="./tests/:/tests/" \
  ghcr.io/grafana/grafana-bench-playwright:v1.0.3 test \
  --service my-api \
  --service-url http://localhost:3000 \
  --service-version 1.0.0 \
  --test-runner playwright \
  --suite-name my-repo/e2e \
  --suite-path ./tests
```

> **Note:** For Playwright troubleshooting (including permission errors with `--with-deps`), see [docs/writing_pw_tests.md#troubleshooting](docs/writing_pw_tests.md#troubleshooting)

## Documentation

- [Getting started & full docs](docs/index.md)
- [Configuration reference](docs/configuration.md)
- [GitHub Actions integration](docs/github_actions.md)
- [Writing K6 tests](docs/writing_k6_api_tests.md)
- [Writing Playwright tests](docs/writing_pw_tests.md)
- [Writing Go tests](docs/writing_go_tests.md)
- [Writing Go benchmarks](docs/writing_go_benchmarks.md)
- [Metrics](docs/metrics.md)
- [Notifications](docs/notifications.md)
- [Architecture](docs/architecture.md)

## Upgrading from v0.6.x?

See the [Migration Guide](docs/migration_v1.md) for step-by-step instructions.

Key changes in v1.0.0:
- `--service` flag is now **required**
- Grafana-specific flags renamed to generic `--service-*` flags
- Credentials passed via `--test-env` (secure passthrough)
- New metric labels: `service`, `suite_name`, `run_stage`

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for build instructions, development setup, and contribution guidelines.

```sh
go build -v ./...
go test -v ./...
make docs   # regenerate CLI docs and update version refs
```
