
# Grafana Bench

Grafana bench is a tool to load test Grafana. It utilizes
https://github.com/grafana/grafana-build and https://github.com/grafana/grafana-api-tests to build and test a grafana on the platform and architecture of your choosing.

## Library

Grafana bench implements a series of functionalities for building, provisioning and testing Grafana.
This building blocks can be used to build testing pipelines

See more info in the [bench library documentation](./bench/README.md)

> !! Using the library is deprecated in favor of using the CLI interface

## CLI interface

Grafana bench provided a CLI interface for executing diverse actions implemented as sub-commands.


To invoke the test runner use the following command from grafana bench's project root directory:

```sh
go run bench.go [options] <sub command> [subcommand options]
```

To get more details of the available options, use the command:

```sh
go run bench.go --help
```

To get more details of the available options for an specific subcommand, use the command:

```sh
go run bench.go <subcommand> --help
```

## Sub Commands

### test

The test subcommand is a wrapper around running grafana-api-tests against a grafana instance.

The test runner executes a set of tests passed as parameter and reports the results.

It can be executed with the following command:

```sh
go run bench.go [options] test [test options] <tests>
```

Supports two kinds of test executions:
* smoke: execute tests and reports failures
* load: execute tests and report execution stats to GCK6

Tests are defined by pointing to a folder. All `.js` files in this folder and its sub-folders will be considered a test and executed

It is also possible to execute a single test by pointing to a `.js` file.

Each test is parameterized via environment variables following [grafana-api-tests conventions](https://github.com/grafana/grafana-api-tests/blob/main/README.md#common-environment-variables).

For load tests, if the `--k6-cloud-output` flag is `true`, the test results will be sent to Grafana Cloud k6. The [GCK6 credentials](https://grafana.com/docs/grafana-cloud/k6/author-run/tokens-and-cli-authentication/) must be provided as environment variables or as arguments to the test runner (`--k6-cloud-project` and `--k6-cloud-token`).

