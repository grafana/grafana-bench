# Grafana Bench

**Bench** provides standardized test execution and observability for any service. It wraps K6, Playwright, Go tests, and Go benchmarks and normalizes results into consistent structured logs and Prometheus metrics.

**Latest Version:** v0.6.11

---

## Quick Links

- **Upgrading from v0.6.x?** See the [Migration Guide](migration_v1.md) 🚀
- **New to Bench?** Start with the [5-minute Quickstart](quickstart.md)
- **Ready to dive in?** Read the [Installation Guide](installation.md)

---

## What is Bench?

Bench is a **labeling strategy** with a CLI to execute your tests in a consistent environment (container), get test output into a consistent format (reporter), plus glue to handle transport, discovery (dashboards), and alerting (metrics / slack) on your test runs.

**Benefits:**

- ✅ **Standardized test execution** across local, CI, and release pipelines
- 📊 **Test observability** with structured logging and Prometheus metrics
- 🔔 **Smart notifications** via Slack with CODEOWNERS integration
- 🎯 **Service-agnostic** — test any HTTP service, gRPC API, or CLI
- 📈 **Load testing** with k6 and k6 Cloud integration

**Components:**

- CLI
- Docker containers (dev/prod, base/playwright - 4 variants)
- [K6 API testing framework](https://github.com/grafana/grafana-api-tests) with support for HTTP, gRPC, and load testing
- Dashboards and metrics via your own Prometheus/Grafana stack
- [versioned libsonnet libraries with argo templates](libsonnet.md)

---

## Why Bench?

Modern applications need testing across many environments, architectures, and configurations. Testing only in CI isn't enough. Bench provides:

### Value Proposition

1. **Standardized test execution** - Run K6, Playwright, or Go tests with a unified interface
2. **Consistent structured output** - All test results normalized to common format
3. **Built-in observability** - Automatic Prometheus metrics and structured logs
4. **Smart notifications** - Slack alerts with team-based routing via CODEOWNERS

### How It Works

Bench is simple: it wraps your testing tools and standardizes the output.

**Architecture:**

- **Test Executors** (K6, Playwright, Go) - Each executor includes a parser for that framework
- **Reporters** - Output results via structured logs, Prometheus metrics, and Slack notifications

Every test runner has two components: a **driver** (executes the tests) and a **parser** (normalizes the output). You can use bench in two ways:

1. **Bench drives and parses** - bench executes your tests and parses the output using a built-in executor
2. **You drive, bench parses** - run your tests yourself and pipe the output to `bench report` with a built-in or custom parser

### Output

Every bench run produces a **structured log line**:

```text
level=info msg="suite complete" suite_name=my-api/smoke service=my-api service_version=1.2.0 run_stage=ci tests_executed=12 tests_passed=11 tests_failed=1 duration_seconds=4.2
```

With `--prometheus-metrics`, bench pushes standard **Prometheus metrics**:

```text
bench_tests_executed{service="my-api",service_version="1.2.0",suite_name="my-api/smoke",run_stage="ci"} 12
bench_tests_passed{service="my-api",service_version="1.2.0",suite_name="my-api/smoke",run_stage="ci"} 11
bench_tests_failed{service="my-api",service_version="1.2.0",suite_name="my-api/smoke",run_stage="ci"} 1
bench_total_duration_seconds{service="my-api",service_version="1.2.0",suite_name="my-api/smoke",run_stage="ci"} 4.2
```

These labels are the same across every test runner, so you can query and compare results uniformly.

### Benefits

- **Portable tests** that run the same everywhere (local → CI → release)
- **Service-agnostic** — works with any service, not just Grafana
- **Unified observability** across all your tests and environments
- **Faster feedback** with standardized reporting and alerting

---

## Getting Started

### 1. Install Bench (2 minutes)

**Docker**:

```sh
# Base image (K6, Go tests, Go benchmarks)
docker pull ghcr.io/grafana/grafana-bench:v1.0.3

# Playwright image (includes Chromium for browser tests)
docker pull ghcr.io/grafana/grafana-bench-playwright:v1.0.3
```

**Binary**:

```sh
go install github.com/grafana/grafana-bench@v0.6.11
```

📖 [Full Installation Guide](installation.md)

### 2. Run Your First Test (5 minutes)

Choose your testing framework:

- **[K6 API Tests](writing_k6_api_tests.md)** - Fast API testing with JavaScript
- **[Playwright Browser Tests](writing_pw_tests.md)** - End-to-end browser testing
- **[Go Tests](writing_go_tests.md)** - Unit and integration tests
- **[Go Benchmarks](writing_go_benchmarks.md)** - Performance benchmarks with metrics export

Or follow the [Complete Quickstart](quickstart.md) for a guided experience.

### 3. Integrate with CI (10 minutes)

Add Bench to your GitHub Actions workflow:

```yaml
- name: Setup Grafana Bench
  uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@281b943dbbdf2a30aa0fd2e5fd07503a49734f44
  with:
    version: 'v1.0.3'

- name: Run Tests
  run: |
    grafana-bench test \
      --suite-name my-repo/tests \
      --suite-path ./tests
```

> **Note on Suite Names:** Use the format `<project>/<test-type>` for clear organization in logs and metrics. Examples: `grafana-bench/go-tests`, `my-plugin/e2e-tests`, `api-service/benchmarks`.

📖 [GitHub Actions Integration Guide](github_actions.md)

---

## Documentation Map

### For New Users

- [Quickstart](quickstart.md) - Get running in 5 minutes
- [Installation](installation.md) - Detailed installation steps
- [Your First Test](first-test.md) - Build your first test step-by-step
- [Validation Guide](validation-guide.md) - Verify your setup

### For Test Authors

- [Writing K6 Tests](writing_k6_api_tests.md) - API testing guide
- [Writing Playwright Tests](writing_pw_tests.md) - Browser testing guide
- [Writing Go Tests](writing_go_tests.md) - Go testing guide
- [Writing Go Benchmarks](writing_go_benchmarks.md) - Performance benchmarking guide
- [Templates](templates.md) - Starter templates and examples
- [Configuration Guide](configuration.md) - Configure bench.yaml

### For DevOps/Platform Engineers

- [GitHub Actions](github_actions.md) - CI integration
- [Libsonnet Pipelines](libsonnet.md) - Deployment pipeline integration
- [Load Testing](load_testing.md) - Performance testing setup
- [Metrics](metrics.md) - Prometheus metrics configuration
- [Notifications](notifications.md) - Slack integration
- [Hosted Grafana Setup](configuring_an_instance.md) - Configure hosted Grafana for e2e tests

### Reference

- [CLI Reference](bench.md) - All commands and flags
- [Architecture](architecture.md) - How bench is structured internally
- [Migration Guide v1.0](migration_v1.md) - Upgrading from v0.6.x
- [Breaking Changes v1.0](breaking_changes_v1.md) - Full changelog of breaking changes
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
- [Glossary](glossary.md) - Key terms and concepts
- [Cheat Sheet](cheat-sheet.md) - Quick reference

### Contributing

- [Contributing Guide](../CONTRIBUTING.md) - Build setup, testing, and how to contribute

---

## Support

- **Bug reports:** [GitHub Issues](https://github.com/grafana/grafana-bench/issues)
- **Roadmap:** [GitHub Project](https://github.com/orgs/grafana/projects/554)

---

## Version & Release Policy

Bench is **semantically versioned** and uses explicit version tags (no `:latest`). This ensures:

- Predictable behavior in CI/CD pipelines
- Controlled upgrade process with clear changelog
- Reduced blast radius of breaking changes

> **Important:** Bench is under active development. We aim for API stability but may introduce breaking changes with version bumps. Always pin specific versions in production workflows.

📖 [Release Notes](https://github.com/grafana/grafana-bench/releases)

---

## Project Status

Bench is under active development. Our basic feature set and value proposition is defined along with a mostly stable API, however, we are trying to move fast and do release breaking changes.

We recognize that as teams are using Bench in CI and for their release pipelines that Bench is critical infrastructure. In order to reduce the blast radius of any change, we provide semantically versioned releases.
