# K6 test Runner

Test runner is a wrapper around running grafana-api-tests against a grafana instance.

It is part of the Grafana Bench project but can be executed independently.

## How it works

To invoke the test runner use the following command from grafana bench's project root directory:

```sh
go run ./cmd/runner [options] <tests>
```

The test runner executes a set of tests passed as parameter and reports the results. 

Supports two kinds of test executions:
* smoke: execute tests and reports failures
* load: execute tests and report execution stats to GCK6

Tests are defined by pointing to a folder. All `.js` files in this folder and its sub-folders will be considered a test and executed

It is also possible to execute a single test by pointing to a `.js` file.

Each test is parameterized via environment variables following [grafana-api-tests conventions](https://github.com/grafana/grafana-api-tests/blob/main/README.md#common-environment-variables).

For load tests, if the `--cloud-output` flag is `true`, the test results will be sent to Grafana Cloud k6. The [GCK6 credentials](https://grafana.com/docs/grafana-cloud/k6/author-run/tokens-and-cli-authentication/) must be provided as environment variables or as arguments to the test runner (`--k6-cloud-project` and `--k6-cloud-token`).

To get more details of the available options, use the command:

```sh
go run ./cmd/runner --help
```
