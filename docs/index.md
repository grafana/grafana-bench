# Grafana Bench

**Latest Version:** v1.0.3

Bench is a CLI tool for standardized test execution and observability. It wraps K6, Playwright, Go tests, and Go benchmarks and normalizes results into consistent structured logs and Prometheus metrics. Every test runner has a driver (executes tests) and a parser (normalizes output) — bench can handle both, or you can run tests yourself and pipe the output to `bench report` with a built-in or custom parser.

**Benefits:**

- ✅ **Standardized test execution** across local, CI, and release pipelines
- 📊 **Test observability** with structured logging and Prometheus metrics
- 🔔 **Smart notifications** via Slack with CODEOWNERS integration
- 🎯 **Service-agnostic** — test any HTTP service, gRPC API, or CLI
- 📈 **Load testing** with k6 and k6 Cloud integration

**Components:**

- CLI
- Docker containers (base and Playwright variants)
- Dashboards and metrics via your own Prometheus/Grafana stack
- [Versioned libsonnet libraries with Argo workflow templates](libsonnet.md)

---

## Quick Links

- **Upgrading from v0.6.x?** See the [Migration Guide](migration_v1.md) 🚀
- **New to Bench?** Start with the [Quickstart](quickstart.md)

---

## Output

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

---

## Documentation Map

### For New Users

- [Quickstart](quickstart.md) - Install and run your first test

### For Test Authors

- [Writing K6 Tests](writing_k6_api_tests.md) - API testing guide
- [Writing Playwright Tests](writing_pw_tests.md) - Browser testing guide
- [Writing Go Tests](writing_go_tests.md) - Go testing guide
- [Writing Go Benchmarks](writing_go_benchmarks.md) - Performance benchmarking guide
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
### Contributing

- [Contributing Guide](contributing.md) - Build setup, testing, and how to contribute

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
