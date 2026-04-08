# Contributing to Grafana Bench

Thank you for your interest in contributing! This guide covers how to build, test, and submit changes. For a deeper overview of how bench is structured, see [Architecture](architecture.md).

## Prerequisites

- [Go](https://go.dev/dl/) 1.26+
- [Docker](https://docs.docker.com/get-docker/) (for container tests)
- [Make](https://www.gnu.org/software/make/)

Install development dependencies:

```sh
make install-deps
```

This installs `jsonnet` and `jsonnetfmt`, required for doc generation.

## Building

```sh
go build -v ./...
```

## Testing

```sh
make test
```

## Running locally

```sh
go run bench.go --help
go run bench.go test --help
```

Or install the binary:

```sh
go install
grafana-bench --help
```

## Updating docs

CLI reference docs are auto-generated from source. After changing any command flags or descriptions, regenerate:

```sh
make docs
```

This also updates version references throughout the documentation. Do not manually edit files prefixed with `bench_` in `docs/` — they are overwritten on every `make docs` run.

The libsonnet library in `libsonnet/` is also generated — run `make libsonnet` to regenerate it locally.

## Project structure

```
bench.go            # Entry point
cmd/                # CLI subcommands (test, report, validate, version)
pkg/
  executor/         # Test runners (k6, playwright, gotest, gobench)
  reporter/         # Output formatters (log, text, prometheus)
  notifier/         # Slack notifications
  config/           # Configuration management
  git/              # Git operations
  metrics/          # Prometheus metrics handling
docs/               # Documentation
```

## Writing a new parser

Parsers live in `pkg/parser/` and convert tool output into a `SuiteRunSummary` that bench can report, log, or push as Prometheus metrics. This is the right place to add support for a new tool — whether it's a linter, scanner, coverage tool, or anything else that produces structured output.

See [`pkg/parser/README.md`](../pkg/parser/README.md) for the recommended workflow, including how to use an LLM to generate both the parser and its metrics from real tool output.

## Submitting changes

1. Fork the repository and create a branch from `main`
2. Make your changes with tests
3. Run `make test` and ensure all tests pass
4. Run `make docs` if you changed any CLI flags or descriptions
5. Open a pull request against `main`

## Reporting issues

Please use [GitHub Issues](https://github.com/grafana/grafana-bench/issues) for bug reports and feature requests.

## Releasing

Releases follow [semantic versioning](https://semver.org/). To create a new release, [create a release on GitHub](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository#creating-a-release) with an appropriately incremented version tag (e.g., `v1.0.7`).

This triggers CI to:
- Build binaries and publish the release with auto-generated notes ([release-binaries.yml](../.github/workflows/release-binaries.yml))
- Update documentation and libsonnet ([release-docs.yml](../.github/workflows/release-docs.yml))

After the release is published, announce it in the [#grafana-bench](https://grafanalabs.enterprise.slack.com/archives/C069CQCLDCG) Slack channel.
