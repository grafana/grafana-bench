#! /usr/bin/env sh

# This script is used for testing and iterating on Bench from inside a container.
# ./run_bech_container.sh tests/dashboards/dashboard_create.js .env
#
# 1. the first argument is path to folder or test file. the file is relative
#    to the volume mounted in the container. 
#    bench binary is located at   /home/bench
#    we assume local tests at     $(PWD)/work/test/suite/dist/tests
#    mounted to                   /home/bench/tests
#    so test path would be        `tests/path/to/my/test.js`
#
# 2. the second is path to .env file. see below for defaults

# source the .env file
if [ "$#" -eq 0 ]; then
  echo "test file is required"
  exit 1
fi

# build a test container
if docker build --platform=linux/amd64 -t grafana-bench-dev .; then
  echo "build succeeded"
else
  echo "build failed"
  exit 1
fi

# source the .env file
if [ "$#" -eq 2 ]; then
  . "$2"
else
  . .env
fi

# location of tests on disk
TEST_LOCATION="$PWD/work/test/suite/dist/tests"

docker run --rm \
  --network host \
  --platform=linux/amd64 \
  --volume="$TEST_LOCATION:/home/bench/tests" \
  grafana-bench-dev test \
    --test-type="smoke" \
    --test-suite "tests/$1"
