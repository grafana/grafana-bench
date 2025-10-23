#! /usr/bin/env sh

# This script is used for testing and iterating on Bench from inside a container.
# ./run_bech_container.sh work/test/suite/dist/tests dashboards/dashboard_create.js .env
#
# 1. the first argument is the path to base folder to find test suites
#    mounted in the container to  /tests.
# 2. the test suite folder or test file, relative to test suite base (see above)
# 3. the third (optional) is path to .env file.

# source the .env file
if [ "$#" -lt 2 ]; then
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

# get path of dotenv file
if [ "$#" -eq 3 ]; then
  DOTENV=$(realpath -e "$3")
  if [ -z "$DOTENV" ]; then
    echo "env file not found"
    exit 1
  fi
else
  DOTENV=$(realpath -e .env)
fi

# build mount portion of .env file
if [ ! -z "$DOTENV" ]; then
  DOTENV_VOLUME="--volume=$DOTENV:/tests/.env"
fi

# location of tests on disk
TESTSUITE_BASE=$(realpath -e "$1")

docker run --rm \
  --network host \
  --platform=linux/amd64 \
  $DOTENV_VOLUME \
  --volume="$TESTSUITE_BASE:/tests" \
  grafana-bench-dev test \
  --test-type="smoke" \
  --test-trigger="rrc" \
  --test-suite-base="tests" \
  --test-suite "$2"
