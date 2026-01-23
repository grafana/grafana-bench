# Bench v1.0.0 Breaking Changes

**Date Started:** January 22, 2026
**Release Target:** v1.0.0
**Related Issue:** [#740](https://github.com/grafana/grafana-bench/issues/740)

## Overview

This document tracks all breaking changes being made for the v1.0.0 release. We're fixing long-standing issues with suite/test run identification and consolidating the data model.

---

## Notes & Decisions

### Run ID Format Decision ✅ IMPLEMENTED
- **Format:** `{stage}-{suiteName}-{timestamp}`
  - `stage`: The run stage (e.g., "rrc", "ci", "local")
  - `suiteName`: The test suite folder path (e.g., "grafana-api-tests/tests"), normalized (slashes→hyphens)
  - `timestamp`: `YYYYDDD-HHMMSS` (year + day of year + time) for uniqueness

### Suite Name Decision
- **Current Problem:** Suite names have unwanted prefixes like "rrc-grafana-api-tests/tests-smoke"
- **Solution:** Use the folder path directly (e.g., "grafana-api-tests/tests")
- **Why:** Cleaner, matches what developers expect, no stage-specific prefixes

### Breaking Changes Philosophy
- Ripping the bandaid off for v1.0.0
- All breaking changes happen at once
- Clear migration path for users

### Service Field Decision ✅ IMPLEMENTED
- **Requirement:** `--service` flag is now **REQUIRED**
- **Purpose:** Identify which service is being tested (e.g., "grafana", "loki", "tempo", "datasources")
- **Logger Changes:** Changed from `service=bench` to `tool=bench, service=<user-specified>`
- **Why:** Makes bench a generic testing reporter that can be used across all Grafana services

### Generic Service Flags Decision 🚧 IN PROGRESS
- **Goal:** Replace Grafana-specific flags with generic service flags
- **Current flags to replace:**
  - `--grafana-url` → `--service-url`
  - `--grafana-version` → `--service-version` (REQUIRED)
  - `--grafana-timeout` → `--service-timeout`
  - `--grafana-admin-user` → REMOVED (only needed for version fetch)
  - `--grafana-admin-password` → REMOVED (only needed for version fetch)

- **New Grafana-specific flag:**
  - `--fetch-grafana-version=user:password` - Fetches version from Grafana API
  - Alternative: `FETCH_GRAFANA_VERSION=user:password` environment variable
  - Mutually exclusive with `--service-version`

- **Version Auto-Detection:** Removed - users must specify version explicitly or use `--fetch-grafana-version`

- **Health Check:**
  - New `--service-health-check` flag to enable health check before running tests
  - Generic TCP-based health check using `--service-url` and `--service-timeout`
  - No auth required (matches Kubernetes health check patterns)
  - Automatically performed before `--fetch-grafana-version` API call
  - Health check logic extracted to `pkg/service/healthcheck.go` (service-agnostic)

- **Why:**
  - Makes bench truly service-agnostic
  - Bench itself doesn't need credentials (tests get them via env vars)
  - Only Grafana-specific convenience is `--fetch-grafana-version` for backwards compat
  - Cleaner, more explicit configuration

---

## Migration Instructions

### BREAKING: --service Flag Now Required

**All bench invocations MUST now include the `--service` flag.**

#### CLI Usage:
```bash
# Before (v0.6.11):
grafana-bench test --test-type smoke --grafana-url http://localhost:3000

# After (v1.0.0):
grafana-bench test --service grafana --test-type smoke --grafana-url http://localhost:3000
```

#### Jsonnet Configuration:
```jsonnet
// In your bench suite configuration
local Suite = benchFunctions.Suite {
  service: 'grafana',  // REQUIRED: Specify the service being tested
  runStage: 'rrc',
  // ...
};
```

**Common service values:**
- `grafana` - For Grafana core testing
- `loki` - For Loki testing
- `tempo` - For Tempo testing
- `datasources` - For datasource plugin testing
- `<your-service>` - Any other service name

### BREAKING: Generic Service Flags (🚧 IN PROGRESS)

**Flag name changes and removals:**

#### Before (v0.6.11):
```bash
grafana-bench test \
  --grafana-url http://localhost:3000 \
  --grafana-admin-user admin \
  --grafana-admin-password admin \
  --grafana-timeout 60s
  # Version auto-detected if not provided
```

#### After (v1.0.0):
```bash
# Option 1: Specify version explicitly (recommended)
grafana-bench test \
  --service grafana \
  --service-url http://localhost:3000 \
  --service-version 11.0.0 \
  --service-timeout 60s

# Option 2: Fetch version from Grafana API
grafana-bench test \
  --service grafana \
  --service-url http://localhost:3000 \
  --fetch-grafana-version=admin:admin \
  --service-timeout 60s

# Option 3: Fetch version using environment variable
export FETCH_GRAFANA_VERSION=admin:admin
grafana-bench test \
  --service grafana \
  --service-url http://localhost:3000 \
  --service-timeout 60s

# Option 4: With health check before running tests
grafana-bench test \
  --service grafana \
  --service-url http://localhost:3000 \
  --service-version 11.0.0 \
  --service-health-check
```

**Key changes:**
- `--grafana-url` → `--service-url`
- `--grafana-version` → `--service-version` (now **REQUIRED**)
- `--grafana-timeout` → `--service-timeout`
- `--grafana-admin-user` and `--grafana-admin-password` → **REMOVED**
  - Only needed for `--fetch-grafana-version`
  - Test credentials passed via environment variables (unchanged)
- **Version auto-detection removed** - must specify explicitly or use `--fetch-grafana-version`
- **New:** `--service-health-check` flag to perform TCP health check before running tests

**For non-Grafana services:**
```bash
# Loki
grafana-bench test \
  --service loki \
  --service-url http://loki:3100 \
  --service-version 2.9.0

# Tempo
grafana-bench test \
  --service tempo \
  --service-url http://tempo:3200 \
  --service-version 2.3.0
```

### For deployment_tools Users

#### Before (v0.6.11):
```jsonnet
// In rrc-bench-suites.libsonnet
local Suite = benchFunctions.Suite {
  runStage: 'rrc',  // This was creating "rrc-" prefixes
  // ...
};
```

**Result:**
- Run ID: `rrc-202622-185217`
- Suite Run Name: `rrc-grafana-api-tests/tests-smoke`

#### After (v1.0.0):
```jsonnet
// In rrc-bench-suites.libsonnet
local Suite = benchFunctions.Suite {
  service: 'grafana',  // REQUIRED: New in v1.0.0
  runStage: 'rrc',     // Still set the stage
  // ...
};
```

**Result:**
- Run ID: `rrc-grafana-api-tests-tests-2026022-185217`
- Suite Name: `grafana-api-tests/tests`
- Service: `grafana` (appears in logs and metrics)

### Query/Dashboard Updates

#### Prometheus Metrics (Mimir-ops datasource)

If you have queries filtering on suite names or run IDs:

**Before:**
```promql
{suite_run="rrc-grafana-api-tests/tests-smoke"}
```

**After:**
```promql
{job="bench", service="grafana", suite_name="grafana-api-tests/tests", run_stage="rrc"}
```

**New labels available:**
- `job="bench"` - Identifies all bench metrics
- `service="<name>"` - The service being tested (grafana, loki, tempo, etc.)
- `suite_name="<path>"` - Clean suite path without stage prefix
- `run_stage="<stage>"` - The run stage (rrc, ci, local)

#### Log Queries (Loki-ops datasource)

**Query for bench logs:**
```logql
{tool="bench", service="grafana"}
```

**New log attributes:**
- `tool="bench"` - Identifies all bench logs
- `service="<name>"` - The service being tested
- `suiteName="<path>"` - The test suite name
- `runStage="<stage>"` - The run stage

#### Dashboards

View bench metrics and logs in Ops Grafana:
- **Dashboards:** https://ops.grafana-ops.net/dashboards/f/d9191c72-9361-4a3b-bb5a-1465e9a8802f/grafanabench
- **Mimir-ops datasource:** Use `{job="bench"}` for Prometheus queries
- **Loki-ops datasource:** Use `{tool="bench"}` for LogQL queries

---

## Architecture & Technical Changes

### Issue #740 & #667 Checklist Status

#### Issue #740 - Suite Identification
- [x] 1. Move attributes to SuiteRun (done previously)
- [x] 2. Remove SuiteRun.SuiteName and SuiteRun.SuiteRevision (done previously)
- [x] 3. Use TestSuite.Name and TestSuite.Revision (done previously)
- [ ] 4. Move metrics to SuiteRun (deferred - not critical for v1.0.0)
- [x] 5. SuiteRun.Name not needed (removed completely)
- [x] 6. Define unique Id for SuiteRun.ID: `{stage}-{suiteName}-{timestamp}` ✅ **DONE**
- [ ] 7. TestRunSummary and SuiteRun summary are result (deferred - needs more investigation)
- [x] 8. Remove SuiteRun.Name `{trigger}-{suiteName}-{testType}` ✅ **DONE**

#### Issue #667 - Service Field Support
- [x] Change service=bench to tool=bench ✅ **DONE**
- [x] Confirm name: chose "service" for the generic service field ✅ **DONE**
- [x] Add --service flag to suite config (REQUIRED field) ✅ **DONE**

#### Issue #666 - Generic Service Support ✅ **DONE**
- [x] Rename GrafanaConfig to ServiceConfig ✅ **DONE**
- [x] Replace Grafana-specific flags with generic service flags ✅ **DONE**
- [x] Add --fetch-grafana-version for backwards compatibility ✅ **DONE**
- [x] Remove auto version detection (require explicit version) ✅ **DONE**
- [x] Deprecated admin credential flags (kept for backward compat, only needed for version fetch) ✅ **DONE**
- [x] Health check already service-agnostic (uses TCP dial) ✅ **DONE**

### Changes Made

#### 1. ID Generation Logic (`pkg/utils/id/id.go`)

**Change:** Updated `Run()` and removed `SuiteRunName()`

**Before:**
```go
func Run(trigger string, time time.Time) string {
    return fmt.Sprintf("%s-%d%d-%d%d%d", trigger, ...)
}

func SuiteRunName(trigger string, suiteName string, testType string) string {
    return fmt.Sprintf("%s-%s-%s", trigger, suiteName, testType)
}
```

**After:**
```go
func Run(stage string, suiteName string, t time.Time) string {
    // Normalize suite name: replace slashes with hyphens
    normalizedSuiteName := strings.ReplaceAll(suiteName, "/", "-")

    return fmt.Sprintf("%s-%s-%d%03d-%02d%02d%02d",
        stage,
        normalizedSuiteName,
        t.Year(),
        t.YearDay(),
        t.Hour(),
        t.Minute(),
        t.Second(),
    )
}

// SuiteRunName removed - use TestSuite.Name directly
```

**Reasoning:**
- Run ID now includes suite context for uniqueness
- Fixed confusing year format: `2026022` (proper zero-padding) instead of `202622`
- Suite names don't need stage/trigger prefix anymore
- Normalizes slashes to hyphens for better compatibility

#### 2. Config Changes (`pkg/config/config.go`)

**Before:**
```go
runId = id.Run(benchConfig.SuiteRun.RunStage, time.Now())

suiteRunName := id.SuiteRunName(
    benchConfig.SuiteRun.RunStage,
    benchConfig.TestSuite.Name,
    benchConfig.Test.Type,
)

return executor.SuiteRun{
    Name: suiteRunName,  // e.g., "rrc-grafana-api-tests/tests-smoke"
    Id: runId,
    // ...
}
```

**After:**
```go
runId = id.Run(
    benchConfig.SuiteRun.RunStage,
    benchConfig.TestSuite.Name,  // Pass suite name for unique ID
    time.Now(),
)

return executor.SuiteRun{
    Name: "",  // Deprecated: Use TestSuite.Name or SuiteRunSummary.SuiteName
    Id: runId,  // e.g., "rrc-grafana-api-tests-tests-2026022-140035"
    // ...
}
```

#### 3. Reporter Changes

**Changes made:**
- `log_reporter.go`: Removed deprecated `suiteRun` field from logs
- `prometheus_reporter.go`: Changed `suite_run` label → `suite_name` label (uses `summary.SuiteName`), added `run_stage` label
- `notification_reporter.go`: Changed to use `suiteRunSummary.SuiteName` instead of `suiteRun.Name`

**Impact:**
- Log queries: Filter on `suiteName` instead of `suiteRun`
- Prometheus queries: Use `suite_name` and `run_stage` labels instead of `suite_run`
- Notifications: No visible change to end users

#### 4. Service Field Changes

**Added Service Field:**
- `pkg/executor/executor.go`: Added `Service string` field to `SuiteRun` struct
- `pkg/config/config.go`:
  - Added `Service string` field to `SuiteRunConfig`
  - Added `--service` flag (REQUIRED)
  - Added validation: returns error if `--service` is empty
  - Updated logger attributes from `service=bench` to `tool=bench, service=<user-value>`

**Reporter Updates:**
- `pkg/reporter/prometheus_reporter.go`: Added `service` label to all metrics
- Logger now includes both `tool="bench"` (the testing tool) and `service="<name>"` (what's being tested)

**Test Updates:**
- All test fixtures updated to include `Service: "grafana"` field
- Ensures tests pass validation

**Reasoning:**
- Separates "what tool is running" (`tool=bench`) from "what's being tested" (`service=grafana`)
- Makes bench a generic testing reporter usable across all Grafana services
- Aligns with multi-tenant/multi-service goals (#666)
- Required field forces teams to be explicit about what they're testing

#### 5. Test Updates

- `config_flag_compatibility_test.go`: Updated to verify `SuiteRun.Id` format instead of deprecated `SuiteRun.Name`
- `log_reporter_test.go`: Removed expectations for deprecated `suiteRun` field
- All reporter tests: Added `Service` field to test fixtures

---

## Files Modified

### Suite Identification (#740)
- [x] `pkg/utils/id/id.go` - ID generation functions
- [x] `pkg/config/config.go` - Updated to use new Run() signature, removed SuiteRun.Name
- [x] `pkg/executor/executor.go` - Removed Name field from SuiteRun struct
- [x] `pkg/reporter/log_reporter.go` - Removed deprecated suiteRun field
- [x] `pkg/reporter/prometheus_reporter.go` - Changed to suite_name + run_stage labels
- [x] `pkg/reporter/notification_reporter.go` - Use summary.SuiteName
- [x] `pkg/config/config_flag_compatibility_test.go` - Updated test expectations
- [x] `pkg/reporter/log_reporter_test.go` - Removed suiteRun expectations
- [x] `pkg/reporter/text_reporter_test.go` - Updated test fixtures
- [x] `pkg/reporter/prometheus_reporter_test.go` - Updated test fixtures
- [x] `pkg/reporter/notification_reporter_test.go` - Updated test fixtures

### Service Field Support (#667)
- [x] `bench.go` - Changed logger from service=bench to tool=bench
- [x] `pkg/config/config.go` - Added Service field, --service flag, validation, and logger attributes
- [x] `pkg/executor/executor.go` - Added Service field to SuiteRun struct
- [x] `pkg/reporter/prometheus_reporter.go` - Added service label
- [x] All test files - Added Service field to test fixtures

### Generic Service Flags (#666) ✅ **DONE**
- [x] `pkg/config/config.go` - Renamed GrafanaConfig to ServiceConfig
- [x] `cmd/test/command.go` - Updated AddGrafanaFlags → AddServiceFlags
- [x] `cmd/report/report.go` - Updated AddGrafanaFlags → AddServiceFlags
- [x] `pkg/config/config_flag_compatibility_test.go` - Updated GrafanaConfig → ServiceConfig
- [x] `pkg/config/config.go` - Added `--service-url`, `--service-timeout`, `--service-version` flags
- [x] `pkg/config/config.go` - Deprecated old `--grafana-*` flags (kept for backward compatibility)
- [x] `pkg/config/config.go` - Added `--fetch-grafana-version` flag with user:password parsing
- [x] `pkg/config/config.go` - Updated BuildSuiteRun to parse FetchVersion and fetch from API if needed
- [x] `pkg/config/config.go` - Removed auto version detection (version now required)
- [x] `pkg/config/config.go` - Added `--service-health-check` flag and health check logic in BuildSuiteRun
- [x] `pkg/service/healthcheck.go` - NEW: Generic service health check implementation
- [x] `pkg/service/healthcheck_test.go` - NEW: Health check tests
- [x] `cmd/test/command.go` - Added health check before running tests (when flag enabled)
- [x] `generators/libsonnet/main.go` - Updated isRequiredStringFlag to use service-url
- [x] `generators/libsonnet/main_test.go` - Updated all tests for new flag names

### Documentation
- [x] `BENCH_V1_BREAKING_CHANGES.md` - Complete breaking changes documentation
- [ ] Update libsonnet codegen (if needed - TBD)
- [ ] Update end-user documentation (README, guides)

---

## Testing Plan

- [x] Unit tests for new ID format - ALL PASS ✅
- [x] Config tests updated and passing ✅
- [x] Reporter tests updated and passing ✅
- [x] All existing tests pass with changes ✅
- [ ] Integration tests with deployment_tools (next step)
- [ ] Verify metrics/logs with new format in RRC
- [ ] Test migration path from v0.6.11 → v1.0.0

---

## Rollout Plan

1. Complete all changes in bench repo
2. Update bench to v1.0.0
3. Test with experimental libsonnet in deployment_tools
4. Update deployment_tools to use v1.0.0
5. Deploy to RRC environment
6. Monitor for issues
7. Update documentation

