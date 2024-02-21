#! /usr/bin/env sh

# This script is used for testing and iterating on Bench from inside a container.
# ./run_bech_container.sh dashboards/dashboard_create.js .env
#
# 1. the first argument is the path to base folder to find test suites
#    mounted in the container to  /home/bench/tests.
# 2. the test suite folder or test file, relative to test suite base (see above)
# 3. the third (optional) is path to .env file.

# source the .env file
if [ "$#" -lt 2]; then
  echo "base dir and test suite are required"
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
if [ "$#" -eq 3 ]; then
  . "$3"
else
  . ./.env
fi

# location of tests on disk
TESTSUITE_BASE=$(readlink -e $1)

docker run --rm \
  --network host \
  --platform=linux/amd64 \
  --volume="$TESTSUITE_BASE:/home/bench/tests" \
  grafana-bench-dev test \
    --test-type="smoke" \
    --test-suite-base="tests" \
    --test-suite "$2"
