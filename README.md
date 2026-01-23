# Grafana Bench

Grafana bench is a tool for testing Grafana.

It's built on top of k6, and [Grafana API Tests](https://github.com/grafana/grafana-api-tests)
to build and test a grafana on the platform and architecture of your choosing.

## 🚀 Upgrading to v1.0.0?

**See [MIGRATION_GUIDE_v1.md](MIGRATION_GUIDE_v1.md)** for step-by-step migration instructions from v0.6.x to v1.0.0.

Key changes in v1.0.0:
- `--service` flag is now **required**
- Grafana-specific flags renamed to generic `--service-*` flags
- Credentials passed via `--test-env` (secure passthrough)
- New metric labels: `service`, `suite_name`, `run_stage`

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

## Running locally

1. build the container you need

`docker build . -f Dockerfile-playwright.dev  -t bench-playwright:dev`

2. start a grafana instance

`docker run --rm grafana/grafana:latest -P 3000:3000`

3. run your tests with bench

If you're checking out tests from a repo include the SUITE_REPO_TOKEN with permissions
to access the repo with tests

```sh
    docker run --rm -e SUITE_REPO_TOKEN --network=host bench-playwright:dev test\
    --service grafana \
    --service-url http://localhost:3000 \
    --service-version 11.0.0 \
    --log-level debug \
    --pw-prepare "yarn install; yarn playwright install" \
    --pw-execute "yarn playwright test" \
    --report-output log \
    --run-stage grafana-bench-ci \
    --suite-path ./CI/playwright \
    --suite-revision main \
    --suite-name grafana-bench/ci/playwright \
    --suite-repo-url https://github.com/grafana/grafana-bench.git \
    --test-runner playwright \
    --test-type smoke
```

> **Note:** For Playwright troubleshooting (including permission errors with `--with-deps`), see [docs/writing_pw_tests.md#troubleshooting](docs/writing_pw_tests.md#troubleshooting)

If you're running local tests, mount the volume inside the container.

### For Local Development (Linux/macOS)

Use the built-in `fixuid` for seamless file permission handling:

```sh
    docker run --rm --network=host --volume="./CI/:/tests/CI/" \
     us-docker.pkg.dev/grafanalabs-dev/docker-grafana-bench-playwright/grafana-bench-playwright:dev-latest test \
      --service grafana \
      --service-url http://localhost:3000 \
      --service-version 11.0.0 \
      --log-level debug \
      --pw-prepare "yarn install; yarn playwright install" \
      --pw-execute "yarn playwright test --grep-invert @performance" \
      --report-output log \
      --run-stage grafana-bench-ci \
      --suite-path ./CI/playwright \
      --suite-revision main \
      --suite-name grafana-bench-dev/ci/playwright \
      --test-runner playwright \
      --test-type smoke
```

### For CI/GitHub Runners (Linux)

Use explicit user mapping to avoid permission issues on large runners:

```sh
    docker run --rm --network=host --volume="./CI/:/tests/CI/" \
     --user "$(id -u):$(id -g)" --env "HOME=/tmp" \
     localhost:5000/grafana-bench-test-playwright-dev:latest test \
      --service grafana \
      --service-url http://localhost:3000 \
      --service-version 11.0.0 \
      --log-level debug \
      --pw-prepare "yarn install; yarn playwright install" \
      --pw-execute "yarn playwright test --grep-invert @performance" \
      --report-output log \
      --run-stage grafana-bench-ci \
      --suite-path ./CI/playwright \
      --suite-revision ${{ github.sha }} \
      --suite-name grafana-bench-dev/ci/playwright \
      --test-runner playwright \
      --test-type smoke
```

### For Windows Development

Docker Desktop handles permissions automatically:

```powershell
    docker run --rm --network=host --volume="./CI/:/tests/CI/" `
     us-docker.pkg.dev/grafanalabs-dev/docker-grafana-bench-playwright/grafana-bench-playwright:dev-latest test `
      --log-level debug `
      --pw-prepare "yarn install; yarn playwright install" `
      --pw-execute "yarn playwright test --grep-invert @performance" `
      --report-format log `
      --run-trigger grafana-bench-ci `
      --suite-path ./CI/playwright `
      --suite-revision main `
      --suite-name grafana-bench-dev/ci/playwright `
      --test-report-format text `
      --test-runner playwright `
      --test-type smoke
```

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

### Setup development dependencies

Before running tests or generating libsonnet libraries, install the required development dependencies:

```sh
make install-deps
```

This installs:
- `jsonnet` - Required for libsonnet generation and testing
- `jsonnetfmt` - Required for formatting generated libsonnet files

### Updating docs

`go run ./gendoc -o docs`
