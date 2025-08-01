#!/bin/sh
set -e

# Production entrypoint - no fixuid
exec grafana-bench "$@"
