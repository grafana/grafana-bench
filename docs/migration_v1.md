# Migration Guide: v0.6.x → v1.0.0

This guide covers the breaking changes in v1.0.0 and how to migrate your existing bench usage.

## Quick Summary

v1.0.0 makes Bench service-agnostic and improves observability:

- **`--service` flag is now REQUIRED** - specify what you're testing
- **Generic service flags** - `--grafana-*` renamed to `--service-*`
- **Secure credentials** - use `--test-env` for passthrough instead of CLI flags
- **Explicit version** - no auto-detection, use `--service-version` or `--fetch-grafana-version`
- **Cleaner metrics** - new labels: `service`, `suite_name`, `run_stage`

## 1. Deprecated Flags (Removed)

These flags have been **completely removed**:

| Removed Flag | Replacement | Notes |
|-------------|-------------|-------|
| `--grafana-admin-user` | `--test-env GRAFANA_ADMIN_USER` | Pass via environment |
| `--grafana-admin-password` | `--test-env GRAFANA_ADMIN_PASSWORD` | Pass via environment |

**Why removed:** Bench doesn't need credentials - only your tests do. Use `--test-env` for secure passthrough.

**Migration:**

```bash
# Before (v0.6.x)
grafana-bench test \
  --grafana-url http://localhost:3000 \
  --grafana-admin-user admin \
  --grafana-admin-password secret

# After (v1.0.0) - Secure passthrough
export GRAFANA_ADMIN_USER=admin
export GRAFANA_ADMIN_PASSWORD=secret

grafana-bench test \
  --service grafana \
  --service-url http://localhost:3000 \
  --service-version 11.0.0 \
  --test-env GRAFANA_ADMIN_USER \
  --test-env GRAFANA_ADMIN_PASSWORD
```

## 2. Renamed Flags

| Old Flag (v0.6.x) | New Flag (v1.0.0) | Required |
|-------------------|-------------------|----------|
| `--grafana-url` | `--service-url` | Yes |
| `--grafana-version` | `--service-version` | Yes* |
| `--grafana-timeout` | `--service-timeout` | No |
| `--test-suite` | `--suite-path` | Yes |
| `--test-suite-base` | `--suite-path` | Yes |
| `--test-suite-name` | `--suite-name` | Yes |
| `--run-trigger` | `--run-stage` | Yes |
| `--report-format` | `--report-output` | No |
| `--test-executor` | `--test-runner` | Yes |
| `--pw-prepare-cmd` | `--pw-prepare` | No |
| `--pw-execute-cmd` | `--pw-execute` | No |

*Required unless using `--fetch-grafana-version`

**Simple find-and-replace:**
```bash
sed -i 's/--grafana-url/--service-url/g' *.sh *.yml
sed -i 's/--grafana-version/--service-version/g' *.sh *.yml
sed -i 's/--test-suite/--suite-path/g' *.sh *.yml
sed -i 's/--run-trigger/--run-stage/g' *.sh *.yml
sed -i 's/--report-format/--report-output/g' *.sh *.yml
sed -i 's/--test-executor/--test-runner/g' *.sh *.yml
```

## 3. Fundamental Changes

### A. `--service` Flag Now Required

**Every bench invocation must specify the service being tested.**

```bash
# Before (v0.6.x)
grafana-bench test --test-type smoke --grafana-url http://localhost:3000

# After (v1.0.0)
grafana-bench test --service grafana --service-url http://localhost:3000 --service-version 11.0.0
```

Common service values: `grafana`, `loki`, `tempo`, `datasources`, or custom names.

### B. Version Auto-Detection Removed

Version is now **required** and must be specified explicitly.

**Option 1: Explicit version (recommended)**
```bash
grafana-bench test \
  --service grafana \
  --service-url http://localhost:3000 \
  --service-version 11.0.0
```

**Option 2: Fetch from Grafana API**
```bash
# Inline credentials
grafana-bench test \
  --service grafana \
  --service-url http://localhost:3000 \
  --fetch-grafana-version admin:admin

# Or via environment variable
export FETCH_GRAFANA_VERSION=admin:admin
grafana-bench test \
  --service grafana \
  --service-url http://localhost:3000
```

### C. Credential Handling

Credentials are now passed to tests via `--test-env`, not to bench itself.

**Two formats:**
- `--test-env KEY` - Passthrough from environment (SECURE - recommended)
- `--test-env KEY=VALUE` - Explicit value (visible in process list)

**Example:**
```bash
# RECOMMENDED: Secure passthrough
export GRAFANA_ADMIN_USER=admin
export GRAFANA_ADMIN_PASSWORD=secret

grafana-bench test \
  --service grafana \
  --service-url http://localhost:3000 \
  --service-version 11.0.0 \
  --test-env GRAFANA_ADMIN_USER \
  --test-env GRAFANA_ADMIN_PASSWORD
```

**In Docker:**
```bash
export GRAFANA_ADMIN_USER=admin
export GRAFANA_ADMIN_PASSWORD=secret

docker run --rm \
  --network=host \
  -e GRAFANA_ADMIN_USER \
  -e GRAFANA_ADMIN_PASSWORD \
  grafana-bench:v1.0.3 test \
  --service grafana \
  --service-url http://localhost:3000 \
  --service-version 11.0.0 \
  --test-env GRAFANA_ADMIN_USER \
  --test-env GRAFANA_ADMIN_PASSWORD
```

**In GitHub Actions:**
```yaml
- name: Run tests
  env:
    GRAFANA_ADMIN_USER: ${{ secrets.GRAFANA_USER }}
    GRAFANA_ADMIN_PASSWORD: ${{ secrets.GRAFANA_PASSWORD }}
  run: |
    grafana-bench test \
      --service grafana \
      --service-url http://localhost:3000 \
      --service-version 12.3.1 \
      --test-env GRAFANA_ADMIN_USER \
      --test-env GRAFANA_ADMIN_PASSWORD
```

### D. Metric Label Changes

Prometheus metrics use new labels for better filtering.

**Before (v0.6.x):**
```promql
bench_tests_passed{suite_run="rrc-grafana-api-tests/tests-smoke"}
```

**After (v1.0.0):**
```promql
bench_tests_passed{
  service="grafana",
  suite_name="grafana-api-tests/tests",
  run_stage="rrc"
}
```

**Available labels:**
- `job="bench"` - Always "bench"
- `service="grafana"` - Service being tested
- `service_version="11.0.0"` - Version of service
- `service_url="http://..."` - URL of service
- `suite_name="path/to/suite"` - Clean suite name
- `run_stage="rrc"` - Execution stage (rrc, ci, local)
- `suite_run_id="..."` - Unique run identifier
- `status="passed"` - Test status

**Update your dashboards:**
- Replace `suite_run` filters with `suite_name` + `run_stage`
- Add `service` filter to separate different services
- Use `service_version` for version-specific filtering

### E. Log Attribute Changes

Loki logs use new attributes.

**Before (v0.6.x):**
```logql
{service="bench"} | json | suiteRun="rrc-grafana-api-tests/tests-smoke"
```

**After (v1.0.0):**
```logql
{tool="bench", service="grafana"}
  | json
  | suiteName="grafana-api-tests/tests"
  | runStage="rrc"
```

**Key changes:**
- `service="bench"` → `tool="bench"`
- Added `service` label for filtering by service type
- `suiteRun` → `suiteName` (cleaner naming)
- Added `runStage` field

### F. Jsonnet Configuration

**Before (v0.6.x):**
```jsonnet
local Suite = benchFunctions.Suite {
  runStage: 'rrc',
  grafanaUrl: 'https://my-stack.grafana.net',
  grafanaVersion: '11.0.0',
  testSuiteName: 'grafana-api-tests/tests',
};
```

**After (v1.0.0):**
```jsonnet
local Suite = benchFunctions.Suite {
  service: 'grafana',              // REQUIRED
  serviceUrl: 'https://my-stack.grafana.net',
  serviceVersion: '11.0.0',
  runStage: 'rrc',
  suiteName: 'grafana-api-tests/tests',
};
```

## Complete Before/After Examples

### CLI Example

**Before (v0.6.x):**
```bash
grafana-bench test \
  --test-runner playwright \
  --grafana-url http://localhost:3000 \
  --grafana-admin-user admin \
  --grafana-admin-password admin \
  --test-suite-base ./CI/playwright \
  --pw-prepare-cmd "yarn install; yarn playwright install" \
  --pw-execute-cmd "yarn test"
```

**After (v1.0.0):**
```bash
export GRAFANA_ADMIN_USER=admin
export GRAFANA_ADMIN_PASSWORD=admin

grafana-bench test \
  --service grafana \
  --service-url http://localhost:3000 \
  --service-version 11.0.0 \
  --test-runner playwright \
  --test-type smoke \
  --suite-path ./CI/playwright \
  --suite-name my-project/ci/playwright \
  --run-stage local \
  --report-output log \
  --pw-prepare "yarn install; yarn playwright install" \
  --pw-execute "yarn test" \
  --test-env GRAFANA_ADMIN_USER \
  --test-env GRAFANA_ADMIN_PASSWORD
```

### GitHub Actions Example

**Before (v0.6.x):**
```yaml
- name: Run tests
  run: |
    grafana-bench test \
      --test-type smoke \
      --grafana-url http://localhost:3000 \
      --test-suite CI/k6
```

**After (v1.0.0):**
```yaml
- name: Run tests
  run: |
    grafana-bench test \
      --service grafana \
      --service-url http://localhost:3000 \
      --service-version 12.3.1 \
      --test-type smoke \
      --suite-path CI/k6 \
      --suite-name ${{ github.repository }}/ci/k6 \
      --suite-revision ${{ github.event.pull_request.head.sha || github.sha }} \
      --run-stage ci \
      --report-output log
```

### Docker Example

**Before (v0.6.x):**
```bash
docker run --rm \
  --network=host \
  --volume="./:/tests/" \
  grafana-bench:v1.0.3 test \
  --grafana-url http://localhost:3000 \
  --test-suite-base /tests
```

**After (v1.0.0):**
```bash
docker run --rm \
  --network=host \
  --volume="./:/tests/" \
  grafana-bench:v1.0.3 test \
  --service grafana \
  --service-url http://localhost:3000 \
  --service-version 11.0.0 \
  --suite-path /tests \
  --suite-name my-project/tests \
  --run-stage local \
  --report-output log
```

## Testing Non-Grafana Services

v1.0.0 is service-agnostic. Test any service:

```bash
# Loki
grafana-bench test \
  --service loki \
  --service-url http://loki:3100 \
  --service-version 2.9.0 \
  --suite-path ./loki-tests

# Tempo
grafana-bench test \
  --service tempo \
  --service-url http://tempo:3200 \
  --service-version 2.3.0 \
  --suite-path ./tempo-tests

# Custom service
grafana-bench test \
  --service my-service \
  --service-url http://custom:8080 \
  --service-version 1.0.0 \
  --suite-path ./custom-tests
```

## Common Issues

### "Error: --service flag is required"
Add `--service grafana` (or your service name).

### "Error: --service-version is required"
Add `--service-version 11.0.0` or use `--fetch-grafana-version admin:admin`.

### Tests fail with authentication errors
Use `--test-env` for credential passthrough:
```bash
export GRAFANA_ADMIN_USER=admin
export GRAFANA_ADMIN_PASSWORD=admin
grafana-bench test --test-env GRAFANA_ADMIN_USER --test-env GRAFANA_ADMIN_PASSWORD ...
```

### Prometheus queries return no data
Update to new labels: `suite_name`, `run_stage`, `service` instead of `suite_run`.

### GitHub Actions: 401 errors with suite-revision
Use PR head SHA: `${{ github.event.pull_request.head.sha || github.sha }}`

## Getting Help

- **Documentation:** [README.md](README.md) and [docs/](docs/)
- **Breaking Changes:** [breaking_changes_v1.md](breaking_changes_v1.md)
- **Issues:** https://github.com/grafana/grafana-bench/issues
- **Examples:** [.github/workflows/](.github/workflows/)
