# Grafana Bench

**Bench** provides test observability and standardized test execution for the Grafana ecosystem.

**Latest Version:** v1.0.0

---

## Quick Links

- **Upgrading from v0.6.x?** See the [Migration Guide](../MIGRATION_GUIDE_v1.md) 🚀
- **New to Bench?** Start with the [5-minute Quickstart](quickstart.md)
- **Ready to dive in?** Read the [Installation Guide](installation.md)
- **Want examples?** Check out [Templates](templates.md)
- **Need help?** See [Troubleshooting](troubleshooting.md)

---

## What is Bench?

Bench is a CLI tool that wraps your testing frameworks (K6, Playwright, Go) and provides:

- ✅ **Standardized test execution** across local, CI, and release pipelines
- 📊 **Test observability** with structured logging and metrics
- 🔔 **Smart notifications** via Slack with CODEOWNERS integration
- 🎯 **Multi-architecture testing** for Grafana's diverse deployment targets
- 📈 **Load testing** with Grafana Cloud K6 integration

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

You either:
1. **Let bench run your tests** - It executes and parses automatically
2. **Pass bench your test output** - It normalizes and reports for you

### Benefits

- **Portable tests** that run the same everywhere (local → CI → release)
- **Team autonomy** with self-service test authoring and feedback
- **Unified observability** across all your tests and environments
- **Faster feedback** with standardized reporting and alerting

For background on testing observability, read: [Delivery Targets and the need for Testing Observability](https://docs.google.com/document/d/1pQoU3sccwayVK_LLzQUnOQLq-0jduH-4FibpdaAztX8/edit?tab=t.0)

---

## Getting Started

### 1. Install Bench (2 minutes)

**Docker** (recommended):

```sh
docker pull us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.6.11
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

Or follow the [Complete Quickstart](quickstart.md) for a guided experience.

### 3. Integrate with CI (10 minutes)

Add Bench to your GitHub Actions workflow:

```yaml
- name: Setup Grafana Bench
  uses: grafana/grafana-bench/.github/actions/setup-grafana-bench@057477c3d586996c1fc3f38772760c34a68d2859
  with:
    version: 'v0.6.11'

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
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
- [Glossary](glossary.md) - Key terms and concepts
- [Cheat Sheet](cheat-sheet.md) - Quick reference

---

## Support

- **Questions?** Ask in [#grafana-bench](https://grafana.slack.com/archives/grafana-bench) Slack channel
- **Bug reports:** [GitHub Issues](https://github.com/grafana/grafana-bench/issues)
- **Roadmap:** [GitHub Project](https://github.com/orgs/grafana/projects/554)
- **Design docs:** [Product DNA](https://docs.google.com/document/d/1rs1RN8UHKAowcQX-cqJ5ilhOnykAMUSoqakWICOGtcY/edit?tab=t.0#heading=h.soj854l81770)

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

Bench is under active development. Our basic feature set and value proposition is defined along with a mostly stable API, however, we are trying to move fast to accommodate teams across Grafana and do release breaking changes.

We recognize that as teams are using Bench in CI and for their release pipelines that Bench is critical infrastructure. In order to reduce the blast radius of any change, we provide semantically versioned releases.
