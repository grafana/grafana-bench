
# Grafana Bench

Grafana bench is a tool for testing Grafana.

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
