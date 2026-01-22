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

---

## Migration Instructions

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
  runStage: 'rrc',  // Still set the stage
  // ...
};
```

**Result:**
- Run ID: `rrc-grafana-api-tests-tests-2026022-185217`
- Suite Name: `grafana-api-tests/tests`

### Query/Dashboard Updates

If you have queries filtering on suite names or run IDs:

**Before:**
```
{suite_name="rrc-grafana-api-tests/tests-smoke"}
```

**After:**
```
{suite_name="grafana-api-tests/tests", run_stage="rrc"}
```

---

## Architecture & Technical Changes

### Issue #740 Checklist Status

- [x] 1. Move attributes to SuiteRun (done previously)
- [x] 2. Remove SuiteRun.SuiteName and SuiteRun.SuiteRevision (done previously)
- [x] 3. Use TestSuite.Name and TestSuite.Revision (done previously)
- [ ] 4. Move metrics to SuiteRun (deferred - not critical for v1.0.0)
- [x] 5. SuiteRun.Name not needed (deprecated - set to empty string)
- [x] 6. Define unique Id for SuiteRun.ID: `{stage}-{suiteName}-{timestamp}` ✅ **DONE**
- [ ] 7. TestRunSummary and SuiteRun summary are result (deferred - needs more investigation)
- [x] 8. Remove SuiteRun.Name `{trigger}-{suiteName}-{testType}` ✅ **DONE**

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

#### 4. Test Updates

- `config_flag_compatibility_test.go`: Updated to verify `SuiteRun.Id` format instead of deprecated `SuiteRun.Name`
- `log_reporter_test.go`: Removed expectations for deprecated `suiteRun` field

---

## Files Modified

- [x] `pkg/utils/id/id.go` - ID generation functions
- [x] `pkg/config/config.go` - Updated to use new Run() signature, deprecated SuiteRun.Name
- [x] `pkg/reporter/log_reporter.go` - Removed deprecated suiteRun field
- [x] `pkg/reporter/prometheus_reporter.go` - Changed to suite_name + run_stage labels
- [x] `pkg/reporter/notification_reporter.go` - Use summary.SuiteName
- [x] `pkg/config/config_flag_compatibility_test.go` - Updated test expectations
- [x] `pkg/reporter/log_reporter_test.go` - Removed suiteRun expectations
- [ ] Update libsonnet codegen (if needed - TBD)
- [ ] Update end-user documentation

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

