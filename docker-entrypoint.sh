#!/bin/sh
set -e

# Check if fixuid is available (dev builds only)
if command -v fixuid >/dev/null 2>&1; then
    eval "$(fixuid)"
fi

exec grafana-bench "$@"
