# Grafana Bench

Bench is a tool for executing e2e tests against an instance of Grafana.

## Quickstart tutorial

1. Prepare github credentials and sign into ghcr repository

Generate a personal access token. Then run the command

echo {YOUR_ACCESS_TOKEN} | docker login ghcr.io -u {YOUR_USERNAME} --password-stdin

1. Pull the container down

`docker pull ghcr.io/grafana/grafana-bench:latest`

1. Download the grafana-api-tests repository

`git clone git@github.com:grafana/grafana-api-tests.git`

1. Build the tests

``` shell
  cd grafana-api-tests
  yarn install
  yarn build
```

1. Boot up an instance of grafana

`docker run -d --name=grafana -p 3000:3000 grafana/grafana`

1. Run the tests

``` shell
    docker run --rm \
     --network host \
     --platform=linux/amd64 \
     --volume="./dist/tests:/home/bench/tests" \
     ghcr.io/grafana/grafana-bench:latest test \
     --test-type="smoke" \
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
     --platform=linux/amd64 \
     --volume="./dist/tests:/home/bench/tests" \
     -e K6_CLOUD_PROJECT_ID="{YOUR_PROJECT_ID}" \
     -e K6_CLOUD_TOKEN="{YOUR_CLOUD_TOKEN}" \
     ghcr.io/grafana/grafana-bench:latest test \
     --k6-cloud-output=true \
     --test-type="smoke" \
     --test-suite-base="tests" \
     --test-suite dashboards
```

## Discover additional bench commands

``` shell
    docker run --rm ghcr.io/grafana/grafana-bench:v0.1.0-rc2 help 
```

## [Write some K6 API tests](writing_k6_api_tests.md)
