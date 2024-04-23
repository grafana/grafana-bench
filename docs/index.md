# Grafana Bench

Bench is a tool for executing e2e tests against an instance of Grafana.

## Quickstart tutorial

1. Pull the container down
`docker pull ghcr.io/grafana/grafana-bench:v0.1.0-rc2`

2. Download some tests
`git clone git@github.com:grafana/grafana-api-tests.git`

3. Build the tests
This is a temporary step that will hopefully removed in the near future as we natively support typescript.

``` shell
  cd grafana-api-tests
  yarn install
  yarn build
```

4. Boot up an instance of grafana
`docker run -d --name=grafana -p 3000:3000 grafana/grafana`

5. Run the tests

``` shell
    docker run --rm \
     --network host \
     --platform=linux/amd64 \
     --volume="./dist/tests:/home/bench/tests" \
     ghcr.io/grafana/grafana-bench:v0.1.0-rc2 test \
     --test-type="smoke" \
     --test-suite-base="tests" \
     --test-suite dashboards
```

6. Report to k6 cloud
Go to <https://ops.grafana-ops.net/a/k6-app/projects>
Click the "Create a new project" button and create a new project
Copy your project ID

Go to <https://ops.grafana-ops.net/a/k6-app/settings/api-token> and copy your api token

Add `K6_CLOUD_PROJECT_ID` and `K6_CLOUD_TOKEN` environment variables to the container and
`--k6-cloud-output=true` to the bench command run in the container.

``` shell
    docker run --rm \
     --network host \
     --platform=linux/amd64 \
     --volume="./dist/tests:/home/bench/tests" \
     -e K6_CLOUD_PROJECT_ID="{YOUR_PROJECT_ID}" \
     -e K6_CLOUD_TOKEN="{YOUR_CLOUD_TOKEN}" \
     ghcr.io/grafana/grafana-bench:v0.1.0-rc2 test \
     --k6-cloud-output=true \
     --test-type="smoke" \
     --test-suite-base="tests" \
     --test-suite dashboards
```
