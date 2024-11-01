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

### Breakdown of the Bench command

1. `docker run --rm` invokes docker. `--rm` tells docker to remove the container when we're done
2. `--network=host` connects the docker container to the same network that the host is on. This is important as the the docker-compose file in the previous step mounts the grafana container to port 3000. So to make grafana accessible from the bench container, we need to connect the bench container to the same network.
3. `--volume="./:/home/bench/tests/"` mounts the current directory of the host machine inside the bench container. In this case, the checkout command from step 1 in the workflow grabs all of the plugin code and puts it in the current directory. So we're mounting everything inside the container in the `/home/bench/tests` directory
4. `us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.3.0 test` says use the bench container tagged with `v0.3.0`. The container specificies the bench binary as the default execution script, so `test` the subcommand and effectively runs `grafana-bench test`
5. `--test-runner "playwright"` tells the test command to use the playwright executor
6. `--test-suite-base "/home/bench/tests/"` specifies the directory we mounted the code in as the directory to execute the test runner from
7. `--pw-prepare-cmd "yarn install --frozen-lockfile; yarn playwright install"` specifies the two commands necessary to configure the e2e tests separated by a `;`. We do not currently support the `&&` operator, so you must use `;`. The first command installs yarn dependencies. The second installs playwright and dependencies
8. `--pw-execute-cmd "yarn e2e"` specifies the command to run the e2e tests.
9. `--test-env-vars "CI=true"` sets an environment variable to be passed to the test executor. Effectively `CI=true grafana-bench test ...`. It is common convention with playwright tests to use the `CI=true` flag
10. `--log-level DEBUG` sets the log level

## Exporting logs to centralized loki database
In development