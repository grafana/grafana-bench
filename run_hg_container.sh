#! /usr/bin/env sh
if docker build --platform=linux/amd64 -t grafana-bench-hg-test:latest -f hg.Dockerfile .; then
  echo "build succeeded"
else
  echo "build failed"
  exit 1
fi

TEST_TYPE=$1
if [ -z "$TEST_TYPE" ]
then
  TEST_TYPE="smoke"
fi

source .env.hg

docker run --rm \
  --platform=linux/amd64 \
  -e K6_CLOUD_PROJECT_ID=$K6_CLOUD_PROJECT_ID \
  -e K6_CLOUD_TOKEN=$K6_CLOUD_TOKEN \
  -e USE_COMPILED_TESTS=$USE_COMPILED_TESTS \
  -e TEST_PATH=$TEST_PATH \
  -e GRAFANA_ADDRESS=$GRAFANA_ADDRESS \
  -e GRAFANA_USER=$GRAFANA_USER \
  -e GRAFANA_PASSWORD=$GRAFANA_PASSWORD \
  -e GRAFANA_TEST_SUITE=$GRAFANA_TEST_SUITE grafana-bench-hg-test:latest $TEST_TYPE
