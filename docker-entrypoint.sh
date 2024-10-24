#!/bin/sh
set -e

eval "$(fixuid)"

exec grafana-bench "$@"
