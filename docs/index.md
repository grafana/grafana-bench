# Grafana Bench

Bench is a tool for executing e2e tests against an instance of Grafana.

## Quickstart tutorial

1. Prepare github credentials and sign into ghcr repository

    Generate a personal access token. Then run the command

    `gh auth token | docker login ghcr.io -u {YOUR_USERNAME} --password-stdin`

1. Pull the container down

    `docker pull ghcr.io/grafana/grafana-bench:latest`

1. Download the grafana-api-tests repository

    `git clone git@github.com:grafana/grafana-api-tests.git`

1. Boot up an instance of grafana

    `docker run -d --name=grafana -p 3000:3000 grafana/grafana`

1. Run tests

    ``` shell
        docker run --rm \
         --network host \
         --volume="./tests:/home/bench/tests" \
         ghcr.io/grafana/grafana-bench:latest test \
         --k6-use-typescript \
         --test-suite-base="tests" \
         --test-suite dashboards
    ```

### Reporting to k6 cloud

1. Go to <https://ops.grafana-ops.net/a/k6-app/projects>
1. Click the "Create a new project" button and create a new project
1. Copy your project ID
1. Go to <https://ops.grafana-ops.net/a/k6-app/settings/api-token> and copy your api token
1. Add `K6_CLOUD_PROJECT_ID` and `K6_CLOUD_TOKEN` environment variables to the container and
`--k6-cloud-output=true` to the bench command run in the container.

``` shell
    docker run --rm \
     --network host \
     --volume="./tests:/home/bench/tests" \
     -e K6_CLOUD_PROJECT_ID="{YOUR_PROJECT_ID}" \
     -e K6_CLOUD_TOKEN="{YOUR_CLOUD_TOKEN}" \
     ghcr.io/grafana/grafana-bench:latest test \
     --k6-cloud-output=true \
     --k6-use-typescript \
     --test-suite-base="tests" \
     --test-suite dashboards
```

### using .env file

Several test runner options can be defined by means of environment variables.

These variables can be set in the `docker run` command as shown in the examples above.

Also,they can be defined in a `.env` file:

```yaml
K6_CLOUD_PROJECT_ID="YOUR_PROJECT_ID" \
K6_CLOUD_TOKEN="YOUR_CLOUD_TOKEN"
GRAFANA_USER="GRAFANA_USER"
GRAFANA_PASSWORD="GRAFANA_PASSWORD" 
```

This file can be passed to the 

``` shell
    docker run --rm \
     --network host \
     --volume="./tests:/home/bench/tests" \
     --volume="./env:/home/bench/.env"  \
     ghcr.io/grafana/grafana-bench:latest test \
     --k6-cloud-output=true \
     --k6-use-typescript \
     --test-suite-base="tests" \
     --test-suite dashboards
```

## Discover additional bench commands

``` shell
    docker run --rm ghcr.io/grafana/grafana-bench:v0.1.0-rc2 help 
```

## Next [Write some K6 API tests](writing_k6_api_tests.md)
