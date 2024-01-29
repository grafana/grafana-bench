#! /usr/bin/env sh

# This script is used for testing and iterating on Bench from inside a container.
# ./run_hg_container.sh tests/dashboards/dashboard_create.js .env
#
# 1. the first argument is path to folder or test file. the file is relative
#    to the volume mounted in the container. 
#    bench binary is located at   /home/bench
#    we assume local tests at     $(PWD)/work/test/suite/dist/tests
#    mounted to                   /home/bench/tests
#    so test path would be        `tests/path/to/my/test.js`
#
# 2. the second is path to .env file. see below for defaults

# build a test container
if docker build --platform=linux/amd64 -t grafana-bench-test .; then
  echo "build succeeded"
else
  echo "build failed"
  exit 1
fi

# source the .env file
if [ "$#" -eq 0 ]; then
  . .env.hg
else
  . "$2"
fi

# location of tests on disk
TEST_LOCATION="$(PWD)/work/test/suite/dist/tests"

# defaults set if not defined in .env file
TEST_TYPE="smoke"
GRAFANA_ADDRESS=${GRAFANA_ADDRESS:-"http://localhost:3000"}
GRAFANA_USER=${GRAFANA_USER:-"admin"}
GRAFANA_PASSWORD=${GRAFANA_PASSWORD:-"admin"}
K6_CLOUD_PROJECT_ID=${K6_CLOUD_PROJECT_ID:-""}
K6_CLOUD_TOKEN=${K6_CLOUD_TOKEN:-""}
K6_CLOUD_OUTPUT=${K6_CLOUD_OUTPUT:-"false"}

docker run --rm \
  --platform=linux/amd64 \
  --volume="$TEST_LOCATION:/home/bench/tests" \
  grafana-bench-test test \
    --test-type="$TEST_TYPE" \
    --grafana-url="$GRAFANA_ADDRESS" \
    --grafana-username="$GRAFANA_USER" \
    --grafana-password="$GRAFANA_PASSWORD" \
    --k6-cloud-project="$K6_CLOUD_PROJECT_ID" \
    --k6-cloud-token="$K6_CLOUD_TOKEN" \
    --k6-cloud-output="$K6_CLOUD_OUTPUT" \
    "$1"
