# Codegen Defaults Refactoring - 2026-01-20

## Summary

Refactored the libsonnet generator to intelligently handle CLI defaults, reducing unnecessary CLI flags while maintaining discoverability of all options.

## Problem

The generated `Suite` template included explicit defaults that matched the CLI defaults (e.g., `gitDriver: 'nanogit'`, `grafanaTimeout: '1m0s'`). This caused unnecessary CLI flags to be added to every bench command, making commands verbose and harder to read.

## Solution

Modified `generators/libsonnet/main.go` to:

1. **Dynamic detection of CLI defaults** - Use `flag.DefaultValue` from cobra instead of static lists
2. **Return zero values for CLI defaults** - Flags with CLI defaults now get `''` or `'0s'` in libsonnet
3. **Production overrides** - Specific flags get non-CLI defaults for production use cases
4. **Maintain discoverability** - All flags remain in the Suite template with their documentation

## Changes Made

### File: `generators/libsonnet/main.go`

Modified the `getLibsonnetDefaultValue()` function:

```go
func getLibsonnetDefaultValue(flag FlagInfo) string {
	// Production overrides: flags that should differ from CLI defaults in libsonnet
	productionOverrides := map[string]string{
		"report-output":          "'log'", // Production needs structured logs (CLI default is 'text')
		"grafana-admin-user":     "''",    // Production uses GRAFANA_ADMIN_USER env var (CLI default is 'admin')
		"grafana-admin-password": "''",    // Production uses GRAFANA_ADMIN_PASSWORD env var (CLI default is 'admin')
	}

	if override, exists := productionOverrides[flag.Name]; exists {
		return override
	}

	// For flags with CLI defaults, return empty/zero so they don't add unnecessary CLI flags
	hasCliDefault := false
	switch flag.Type {
	case "bool":
		hasCliDefault = flag.DefaultValue == "true"
	case "string":
		hasCliDefault = flag.DefaultValue != "" && !isRequiredStringFlag(flag.Name)
	case "duration":
		hasCliDefault = flag.DefaultValue != "" && flag.DefaultValue != "0s"
	case "int":
		hasCliDefault = flag.DefaultValue != "" && flag.DefaultValue != "0"
	}

	if hasCliDefault {
		// Return zero value to avoid generating CLI flag
		switch flag.Type {
		case "bool":
			return "false"
		case "duration":
			return "'0s'"
		case "int":
			return "0"
		default:
			return "''"
		}
	}

	// Standard handling for flags without CLI defaults...
}
```

## Production Overrides

Three flags get special production defaults that differ from CLI:

| Flag | CLI Default | Libsonnet Default | Reason |
|------|-------------|-------------------|--------|
| `report-output` | `'text'` | `'log'` | Production needs structured logs |
| `grafana-admin-user` | `'admin'` | `''` | Production uses `GRAFANA_ADMIN_USER` env var |
| `grafana-admin-password` | `'admin'` | `''` | Production uses `GRAFANA_ADMIN_PASSWORD` env var |

## CLI Defaults Now Empty in Libsonnet

These flags now have zero/empty values in the generated Suite, avoiding unnecessary CLI flags:

| Flag | Old Libsonnet Default | New Libsonnet Default | CLI Default |
|------|----------------------|----------------------|-------------|
| `gitDriver` | `'nanogit'` | `''` | `'nanogit'` |
| `grafanaTimeout` | `'1m0s'` | `'0s'` | `'1m0s'` |
| `suiteBase` | `'.'` | `''` | `'.'` |
| `testRunner` | `'k6'` | `''` | `'k6'` |
| `testType` | `'smoke'` | `''` | `'smoke'` |
| `runStage` | `'local'` | `''` | `'local'` |
| `slackCodeownersMapping` | `'codeowners-mapping.yaml'` | `''` | `'codeowners-mapping.yaml'` |

## Benefits

1. **Cleaner CLI commands** - No unnecessary flags when using defaults
2. **Automatic adaptation** - Changes to CLI defaults automatically reflected in codegen
3. **Discoverability maintained** - All flags still visible in Suite with documentation
4. **User override support** - Users can still set any flag explicitly in their Suite definitions

## Before/After Example

### Before (v0.6.11)
```jsonnet
Suite:: {
  gitDriver: 'nanogit',
  grafanaAdminUser: 'admin',
  grafanaAdminPassword: 'admin',
  grafanaTimeout: '1m0s',
  reportOutput: 'text',
  runStage: 'local',
  testRunner: 'k6',
  testType: 'smoke',
  // ...
}
```

Generated command included unnecessary flags:
```bash
grafana-bench test --grafana-url ... --suite-path ... --git-driver nanogit --grafana-admin-user admin --grafana-admin-password admin --grafana-timeout 1m0s --report-output text --run-stage local --test-runner k6 --test-type smoke
```

### After (new codegen)
```jsonnet
Suite:: {
  gitDriver: '',
  grafanaAdminUser: '',
  grafanaAdminPassword: '',
  grafanaTimeout: '0s',
  reportOutput: 'log',  // Production override
  runStage: '',
  testRunner: '',
  testType: '',
  // ...
}
```

Generated command is much cleaner:
```bash
grafana-bench test --grafana-url ... --suite-path ... --log-level info --test-report-format log --report-output log
```

## Testing

Tested by regenerating test-codegen version:
```bash
cd bench
go run ./generators/libsonnet/main.go generate --target-version test-codegen -o /tmp/bench-test
```

Verified output in `/tmp/bench-test/test-codegen/main.libsonnet` shows:
- All CLI-default flags are now empty/zero
- Production overrides are applied correctly
- All flags remain documented with comments
- Required flags still have `error 'must define ...'` defaults

## RRC Workflow Fix

Also updated RRC suite configuration to use the new pattern:

**File: `ksonnet/environments/hosted-grafana-cd/rrc-bench-suites.libsonnet`**

```jsonnet
local Suite = benchFunctions.Suite {
  // Override defaults to use environment variables instead of CLI flags
  grafanaAdminUser: '',
  grafanaAdminPassword: '',

  // Use structured log output for production
  reportOutput: 'log',

  // ... rest of config
}
```

This ensures:
1. Grafana credentials come from `GRAFANA_ADMIN_USER`/`GRAFANA_ADMIN_PASSWORD` env vars (not CLI flags)
2. Structured logs are used for production (not human-readable text)
3. Other defaults rely on CLI to avoid verbose commands

## Future Versions

When new CLI flags are added or defaults change in `grafana-bench`:
1. Codegen automatically detects new defaults via `flag.DefaultValue`
2. No manual updates needed to libsonnet generator
3. Just regenerate versions and they'll have correct behavior

## Additional Fix: testEnv Bug

### Problem
The generated `buildScript()` function had a bug in handling `testEnv`:

```jsonnet
// OLD - BUGGY
local test_env_vars = [std.split(env, '=')[0] + '="$"' + std.split(env, '=')[0] for env in bench_options.testEnv];
```

This took `testEnv: ['CI=1', 'GRAFANA_BENCH=1']` and converted it to `CI="$CI",GRAFANA_BENCH="$GRAFANA_BENCH"`, trying to reference container environment variables that don't exist.

### Solution
Fixed template and code generation to pass `testEnv` values directly:

**File: `generators/libsonnet/tmpl/main.libsonnet.tmpl`**
- Removed buggy `test_env_vars` variable manipulation
- Added comment explaining that testEnv contains key=value pairs

**File: `generators/libsonnet/main.go`** (line 672-675)
```go
// Special handling for test-env: testEnv in libsonnet is an array of "key=value" strings for simplicity
// Users write: testEnv: ['CI=1', 'VAR=$ENV_VAR'] and we pass them directly to --test-env
if flag.Name == "test-env" {
    return "+ (if bench_options.testEnv != [] then ['--test-env', std.join(',', bench_options.testEnv)] else [])"
}
```

### Result
Now `testEnv: ['CI=1', 'GRAFANA_BENCH=1']` correctly generates:
```bash
--test-env CI=1,GRAFANA_BENCH=1
```

Bench handles these correctly via `os.ExpandEnv()`, so users can still reference env vars with `$VAR` syntax if needed.

## Next Steps

1. ✅ Test codegen - DONE
2. ✅ Document changes - DONE
3. ✅ Fix testEnv bug - DONE
4. 🔲 Regenerate all versions in deployment_tools with new codegen
5. 🔲 Update bench CI to use new codegen approach
