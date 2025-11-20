# Grafana Bench Libsonnet Generator

This directory contains the libsonnet generator for the grafana-bench project. The generator creates Jsonnet libraries that can be used in deployment workflows to execute bench tests with proper configuration.

## Overview

The generator uses a subcommand approach to create two types of libsonnet files:

1. **Main Functions** (`generate` subcommand) - Creates the core libsonnet function for a specific version
2. **Version Mapping** (`versions` subcommand) - Creates a version mapping file that imports multiple versions

## Architecture

### Subcommands

#### `generate` - Generate Main Functions
```bash
go run generators/libsonnet generate -o <output-dir> [-version <version>]
```

Creates:
- `main.libsonnet` - Core function with all CLI flags as configurable options
- `main_test.jsonnet` - Unit tests for the generated functions

#### `versions` - Generate Version Mapping  
```bash
go run generators/libsonnet versions --versions "v1.0.0,v1.1.0,experimental" -o <output-dir>
```

Creates:
- `versions.libsonnet` - Dynamic version mapping with import statements
- `versions_test.jsonnet` - Tests for version mapping functionality

### Template System

Templates are stored in `tmpl/` directory:
- `main.libsonnet.tmpl` - Template for main function generation
- `main_test.jsonnet.tmpl` - Template for main function tests
- `versions.libsonnet.tmpl` - Template for version mapping
- `versions_test.jsonnet.tmpl` - Template for version mapping tests

### Code Generation Process

1. **Flag Extraction**: Uses Cobra command introspection to extract all CLI flags from the `test` command
2. **Template Data**: Converts flags to template data with proper type handling and defaults
3. **Template Rendering**: Renders templates with extracted data
4. **Formatting**: Uses `jsonnetfmt` to format generated files
5. **Linting**: Uses `jsonnet-lint` to validate generated code

## Usage

### Development Workflow

```bash
# Generate libsonnet for development (uses latest git tag)
make libsonnet

# Generate libsonnet for specific version  
make libsonnet-release VERSION=v1.2.3

# Test just the versions subcommand
make libsonnet-test-versions
```

### Manual Generation

```bash
# Generate main functions for a specific version
go run generators/libsonnet generate -o output/ -version v1.2.3

# Generate versions mapping
go run generators/libsonnet versions --versions "experimental,v1.0.0,v1.1.0" -o output/

# Show help
go run generators/libsonnet help
```

## Deployment Integration

### CI/CD Pipeline

The generator is integrated into GitHub Actions workflows:

1. **Local Generation**: `make libsonnet-release VERSION=$VERSION`
2. **Version Mapping**: `go run generators/libsonnet versions --versions "experimental,$VERSION" -o libsonnet`
3. **Docker Deployment**: Mount entire repo and copy files to deployment_tools
4. **Dynamic Discovery**: Deployment tools discover existing versions and regenerate mapping

### File Structure in deployment_tools

```
ksonnet/lib/bench/
├── experimental/
│   ├── main.libsonnet
│   └── main_test.jsonnet  
├── v1.0.0/
│   ├── main.libsonnet
│   └── main_test.jsonnet
├── versions.libsonnet      # Dynamic mapping of all versions
└── versions_test.jsonnet   # Tests for version mapping
```

## Generated Code Structure

### main.libsonnet

```jsonnet
function(name) {
  withBenchTest(grafanaURL, suite):: self {
    local bench_options = {
      // All CLI flags as configurable options with defaults
      suitePath: error 'must define suite path',
      testRunner: 'k6',
      testType: 'smoke',
      // ... all other flags
    } + suite,
    
    // Generates script array for Argo workflows
    local script = [
      'grafana-bench', 'test',
      '--grafana-url', grafanaURL,
      // ... conditional flags based on options
    ],
    
    parameters+: {
      script: std.join(' ', script),
      image: selectedImage + ':' + bench_options.benchRevision,
    },
  },
}
```

### versions.libsonnet  

```jsonnet
local versions = {
  experimental: import 'experimental/main.libsonnet',
  'v1.0.0': import 'v1.0.0/main.libsonnet',
};

{
  getBenchFunctions(version):: versions[version],
  getAvailableVersions():: std.objectFields(versions),
  hasVersion(version):: std.objectHas(versions, version),
}
```

## Quality Assurance

### Testing

- **Unit Tests**: Every generated file includes comprehensive unit tests
- **Linting**: All generated files pass `jsonnet-lint` validation  
- **Formatting**: All files are formatted with `jsonnetfmt`
- **Integration**: Tests verify actual functionality works end-to-end

### Makefile Integration

```bash
make libsonnet                    # Full workflow: versions test + main generation + testing
make libsonnet-test-versions      # Test versions subcommand with mock versions
make libsonnet-release VERSION=X # Generate for specific version
```

The workflow ensures:
1. Versions subcommand generates valid files and passes tests
2. Main generation creates properly formatted and linted code  
3. All generated functions work correctly in practice

## Flag Handling

### Supported Types
- `bool` - Conditional flag inclusion
- `string` - String values with escaping where needed
- `stringArray/stringSlice` - Multiple values with flattenArrays
- `stringToString` - Key=value pairs (special handling for test-env)
- `int` - Numeric values converted to strings
- `duration` - Duration strings

### Special Cases
- **Required flags**: `grafana-url`, `suite-path` - Generate `error` defaults
- **Escaped strings**: `pw-execute`, `pw-prepare` - Use `std.escapeStringBash`  
- **URL escaping**: `run-dashboard` - Use custom `escapeString` function
- **Test environment**: `test-env` - Special array-to-comma-separated handling

### Deprecated Flags
The generator automatically skips deprecated and legacy flags to keep the libsonnet clean and focused on current functionality.

## Examples

### Using Generated Libsonnet

```jsonnet
// Import version mapping
local benchVersions = import 'bench/versions.libsonnet';

// Get functions for specific version  
local benchFunctions = benchVersions.getBenchFunctions('v1.2.3');

// Create a test configuration
local benchTest = benchFunctions('my-test')
  .withBenchTest('http://grafana:3000', {
    suitePath: 'CI/k6',
    testRunner: 'k6', 
    testType: 'smoke',
    prometheusMetrics: true,
    prometheusUrl: 'http://prometheus:9090',
  });

// Use in Argo workflow
{
  spec: {
    templates: [{
      name: 'bench-test',
      container: {
        image: benchTest.parameters.image,
        command: ['sh', '-c', benchTest.parameters.script],
      },
    }],
  },
}
```

This provides a clean, type-safe way to configure bench tests with all CLI options available as libsonnet parameters.