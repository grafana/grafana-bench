# Agent Instructions

This file provides guidance to AI coding agents (Claude Code, Codex, etc.) when working with this repository.

## Overview

Grafana Bench is a Go-based CLI tool for standardized test execution and observability. It normalizes test output into structured logs and Prometheus metrics. Every test runner has a driver (executes tests) and a parser (normalizes output) — bench can handle both, or you can run tests yourself and pipe the output to bench with a built-in or custom parser. It is service-agnostic and works with any testing service provided there is an executor and a parser.

## Build and Test Commands

### Core Commands
- **Build**: `go build -v ./...`
- **Test**: `make test`
- **Install**: `go install`
- **Run locally**: `go run bench.go [command] [flags]`

### Make Targets
- **Test**: `make test`
- **Update docs**: `make docs` (regenerates CLI docs and updates version refs)
- **Generate libsonnet**: `make libsonnet`

### Test Execution Examples
```bash
# Run K6 smoke tests against a local service
grafana-bench test \
  --service my-api \
  --service-url http://localhost:3000 \
  --service-version 1.0.0 \
  --test-runner k6 \
  --suite-name my-repo/smoke \
  --suite-path ./tests/k6 \
  --run-stage ci

# Run Go benchmarks with Prometheus metrics
grafana-bench test \
  --service bench \
  --service-version v1.0.0 \
  --test-runner gobench \
  --suite-path ./benchmarks \
  --gobench-pattern "BenchmarkAPI" \
  --gobench-time 10s \
  --prometheus-metrics \
  --run-stage ci

# Validate Slack permissions
grafana-bench validate --check-slack-permissions
```

## Architecture

### CLI Structure
- **Entry point**: `bench.go` - main executable that initializes logger and root command
- **Commands**: `cmd/` directory contains all subcommands:
  - `test/` - Test execution with K6, Playwright, Go, and gobench support
  - `report/` - Test result reporting and formatting
  - `validate/` - Configuration and permission validation
  - `version/` - Version information

### Core Packages
- **executor/**: Test execution engines for different frameworks:
  - `gotest/` - Go test execution and parsing
  - `gobench/` - Go benchmark execution with performance metrics
  - `k6/` - K6 JavaScript test execution
  - `playwright/` - Playwright browser test execution
- **reporter/**: Multiple output formats (log, text, Prometheus metrics)
- **notifier/**: Slack integration with CODEOWNERS mapping
- **config/**: Configuration management with Viper
- **git/**: Git repository operations and revision detection
- **metrics/**: Prometheus metrics handling and validation

### Test Executors and Reporting Support
Look in pkg/executors to see a current list of testing tools we can execute.
Look in pkg/parsers to see a current list of parsers we can support

## Configuration

### Environment Variables
- `BENCH_LOG_LEVEL`: Override log level (ERROR, WARN, INFO, DEBUG)
- `SLACK_TOKEN`: Required for Slack notifications
- `TEST_SUITE_REPO_TOKEN`: GitHub token for repository access
- `PROMETHEUS_PASSWORD`: Token for Prometheus remote write

### Configuration Files
- `bench.yaml`: Main configuration file (default)
- `.env`: Environment variables (loaded automatically if present)
- `codeowners-mapping.yaml`: Maps GitHub teams to Slack channels for notifications

## Testing Strategy

- Unit tests for individual components (parsers, executors, utilities)
- Integration tests for full workflow scenarios
- Test data in `testdata/` directories for parser validation

## Development Notes

- Uses Cobra for CLI framework and Viper for configuration
- Docker images published to `ghcr.io/grafana/grafana-bench` and `ghcr.io/grafana/grafana-bench-playwright`
- Auto-generated docs in `docs/bench*.md` — do not edit directly, run `make docs` instead
- When writing or editing documentation, follow the conventions in [DOCS_STYLE_GUIDE.md](DOCS_STYLE_GUIDE.md)

---

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

1. **File issues for remaining work** - Use [GitHub Issues](https://github.com/grafana/grafana-bench/issues) for anything that needs follow-up
2. **Run quality gates** (if code changed) - `make test`, `go build -v ./...`; run `make docs` and `make libsonnet` if CLI flags or libsonnet changed
3. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   git push
   git status  # MUST show "up to date with origin"
   ```
4. **Verify** - All changes committed AND pushed
