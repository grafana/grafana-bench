#!/bin/sh
set -e

# Development entrypoint with fixuid for local volume mounting
eval "$(fixuid)"

exec grafana-bench "$@"