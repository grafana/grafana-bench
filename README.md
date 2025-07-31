# Grafana Bench

Grafana bench is a tool for testing Grafana.

It's built on top of k6, and [Grafana API Tests](https://github.com/grafana/grafana-api-tests) to build and test a grafana on the platform and architecture of your choosing.

## Docs

We keep updated docs in the docs/ directory which is published to
[enghub](https://enghub.grafana-ops.net/docs/default/component/grafana-bench).

## A note on the Bench docker images

You can install the binary or run the docker image. We generally prefer to run
the docker image as that's what runs the tests in production CI.

There are two images, grafana-bench and grafana-bench-playwright which each have
a dev and prod version. The ONLY difference between the dev and prod images is
we configure the uid's so that you can mount a volume to the container and run
tests for local development. You don't need to use this image if you are
having bench check out tests from a repo.

* grafana-bench is a minimal image with grafana-bench and k6

* grafana-bench-playwright is for browser testing and includes all the
playwright dependencies preinstalled for you preinstalled for you.

**NOTE**
Due to the way dependencies are configured the home directory and user for the grafana-bench-playwright image needs to
be `pwuser` rather than `bench`. So if you would normally put files in `/home/bench/tests` you'll put those in
`/home/pwuser/tests` instead

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

See [documentation online](https://github.com/grafana/grafana-bench/blob/main/docs/index.md)

## Contributing

### Updating docs

`go run ./gendoc -o docs`
