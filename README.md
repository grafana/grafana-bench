
# Grafana Bench

Grafana bench is a tool to build, provision, and test Grafana.

It's built on top of k6, and [Grafana API Tests](https://github.com/grafana/grafana-api-tests) to build and test a grafana on the platform and architecture of your choosing.


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

The test subcommand is a wrapper for running k6 tests against a grafana instance.

The test runner executes a set of tests passed as parameter and reports the results.

> Executing tests requires to have `k6` binary installed in the machine that executes the `bench test` command.
> It can be installed it using the `go install go.k6.io/k6@latest` command or following any of the other suggested [installation procedures](https://grafana.com/docs/k6/latest/get-started/installation/)

It can be executed with the following command:

```sh
go run bench.go [options] test [test options] --test-suite <test-suite>
```

Supports two kinds of test executions defined by the `test-type` option:
* smoke: execute tests and reports failures (default)
* load: execute tests and report execution stats to GCK6

The tests to be executed are defined by the `--test-suite` option. It can be a directory or a single `.js` file.
If a directory is specified, all `.js` files in the directory and its sub-directories will be considered a test and executed.

Each test is parameterized via environment variables following [grafana-api-tests conventions](https://github.com/grafana/grafana-api-tests/blob/main/README.md#common-environment-variables).

For load tests, if the `--k6-cloud-output` flag is `true`, the test results will be sent to Grafana Cloud k6. The [GCK6 credentials](https://grafana.com/docs/grafana-cloud/k6/author-run/tokens-and-cli-authentication/) must be provided as environment variables or as arguments to the test runner (`--k6-cloud-project` and `--k6-cloud-token`).
