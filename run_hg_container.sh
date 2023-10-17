#! /usr/bin/env sh
if docker build --platform=linux/amd64 -t grafana-bench-test .; then
  echo "build succeeded"
else
  echo "build failed"
  exit 1
fi


source .env.hg

USE_COMPILED_TESTS=true
TEST_PATH=/home/bench/tests
TEST_TYPE=${TEST_TYPE:-"smoke"}

GRAFANA_ADDRESS=${GRAFANA_ADDRESS:-"http://localhost:3000"}
GRAFANA_USER=${GRAFANA_USER:-"admin"}
GRAFANA_PASSWORD=${GRAFANA_PASSWORD:-"admin"}

docker run --rm \
  --platform=linux/amd64 \
  --volume=$(PWD)/work/test/suite/dist/tests:/home/bench/tests \
  -e K6_CLOUD_PROJECT_ID=$K6_CLOUD_PROJECT_ID \
  -e K6_CLOUD_TOKEN=$K6_CLOUD_TOKEN \
  -e USE_COMPILED_TESTS=$USE_COMPILED_TESTS \
  -e TEST_PATH=$TEST_PATH \
  -e GRAFANA_TEST_SUITE=$GRAFANA_TEST_SUITE grafana-bench-test $TEST_TYPE $GRAFANA_ADDRESS $GRAFANA_USER $GRAFANA_PASSWORD all
