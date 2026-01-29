# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Grafana Bench is a Go-based CLI tool for testing Grafana across multiple architectures and configurations. It provides test observability and standardized test execution/reporting for the Grafana ecosystem, supporting K6 API tests, Playwright browser tests, Go tests, and Go benchmarks.

## Build and Test Commands

### Core Commands
- **Build**: `go build -v ./...`
- **Test**: `go test -v ./...`
- **Install**: `go install`
- **Run locally**: `go run bench.go [command] [flags]`

### Documentation Generation
- **Update docs**: `go run ./gendoc -o docs`

### Test Execution Examples
```bash
# Run smoke tests against local Grafana
grafana-bench test \
  --test-type smoke \
  --grafana-url http://localhost:3000 \
  --test-suite CI/k6 \
  --log-level info

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
  - `test/` - Test execution with K6 and Playwright support
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

### Test Framework Support
1. **K6 API Tests**: JavaScript-based API testing in `CI/k6/`
2. **Playwright Browser Tests**: End-to-end browser testing in `CI/playwright/`
3. **Go Tests**: Standard Go unit/integration tests throughout codebase
4. **Go Benchmarks**: Performance benchmarks with metrics export to Prometheus

## Configuration

### Environment Variables
- `BENCH_LOG_LEVEL`: Override log level (ERROR, WARN, INFO, DEBUG)
- `SLACK_TOKEN`: Required for Slack notifications
- `TEST_SUITE_REPO_TOKEN`: GitHub token for repository access

### Configuration Files
- `bench.yaml`: Main configuration file (default)
- `.env`: Environment variables (loaded automatically if present)
- `codeowners-mapping.yaml`: Maps GitHub teams to Slack channels for notifications

## Testing Strategy

The codebase uses a multi-layered testing approach:
- Unit tests for individual components (parsers, executors, utilities)
- Integration tests for full workflow scenarios
- E2E tests in CI pipeline with live Grafana instance
- Test data in `testdata/` directories for parser validation

## Development Notes

- Uses Cobra for CLI framework and Viper for configuration
- Logging via custom logger package with structured output
- Git operations for test suite management and revision tracking
- Docker support with separate images for slim and Playwright variants
- CI runs tests against live Grafana instance on port 3000