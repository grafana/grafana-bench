# Running bench in Github Action CI
One of the key goals of Bench is to provide portability. We do this by making bench part of the development workflow- running tests locally during development, in CI as part of the SDLC, and finally in [release pipelines](libsonnet.md) as part of release.

## Datasource workflow example
This is a live example of a workflow from the [clickhouse datasource](https://github.com/grafana/clickhouse-datasource/blob/main/.github/workflows/grafana-bench.yml).


```yaml
name: Grafana Bench
on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  test:
    timeout-minutes: 60
    runs-on: ubuntu-latest
    steps:
    # Plugin setup
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'yarn'

      - uses: actions/setup-go@v5
        with:
          go-version: 'stable'

      - name: Build backend
        uses: magefile/mage-action@v3
        with:
          args: buildAll
          version: latest

      - name: Install frontend dependencies
        run: yarn install --frozen-lockfile

      - name: Build frontend
        run: yarn build
        env:
          NODE_OPTIONS: '--max_old_space_size=4096'

      - name: Install and run Docker Compose
        uses: hoverkraft-tech/compose-action@v2.0.2
        with:
          compose-file: './docker-compose.yml'
    # Bench
      - name: Ensure Grafana is running
        run: |
          curl http://localhost:3000

      - name: Run Grafana Bench tests
        run: |
          docker run --rm \
            --network=host \
            --volume="./:/home/bench/tests/" \
            us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.3.0 test \
            --test-runner "playwright" \
            --test-suite-base "/home/bench/tests/" \
            --grafana-url "http://localhost:3000" \
            --pw-prepare-cmd "yarn install --frozen-lockfile; yarn playwright install" \
            --pw-execute-cmd "yarn e2e" \
            --test-env-vars "CI=true" \
            --log-level DEBUG 
```

### What's happening
We configure the tests to run on every PR and merge to Main. This gives us early and often feedback.


```yaml
on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]
```

This workflow first configures the plugin:
1. checkout the code
2. install node
3. install go
4. build the backend
5. install frontend dependencies
6. install docker-compose to run the plugin

Then we run Bench:

7. ensure the container is running
8. invoke bench against the container

## Exporting logs to centralized loki database
In development