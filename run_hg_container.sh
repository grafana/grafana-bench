#! /usr/bin/env sh
if docker build --platform=linux/amd64 -t grafana-bench:latest .; then
  echo "build succeeded"
else
  echo "build failed"
  exit 1
fi

source .env.hg

docker run --rm \
  --platform=linux/amd64 \
  -e GCP_KEY_BASE64=$GCP_KEY_BASE64 \
  -e K6_CLOUD_PROJECT_ID=$K6_CLOUD_PROJECT_ID \
  -e K6_CLOUD_TOKEN=$K6_CLOUD_TOKEN \
  -e GRAFANA_VERSION=$GRAFANA_VERSION \
  -e GRAFANA_URL=$GRAFANA_URL \
  -e GRAFANA_PORT=$GRAFANA_PORT \
  -e GRAFANA_USER=$GRAFANA_USER \
  -e GRAFANA_PASSWORD=$GRAFANA_PASSWORD \
  -e GRAFANA_TEST_SUITE=$GRAFANA_TEST_SUITE grafana-bench:latest

