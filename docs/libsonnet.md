# Libsonnet Library

`grafana-bench` offers a libsonnet library that facilitates the execution of a bench test suite in ArgoCD workflows.

## Example

Import the version mapping and configure a suite using the generated `Suite` template:

```js
// Import the version mapping
local benchVersions = import 'bench/versions.libsonnet';

// Get functions for a specific version
local bench = benchVersions.getBenchFunctions('v1.0.3');

// Define a suite by extending the Suite template
local suite = bench.Suite + {
  service: 'my-service',
  serviceUrl: 'http://my-service:3000',
  serviceVersion: '1.2.0',
  testRunner: 'k6',
  testType: 'smoke',
  suiteName: 'my-service/smoke',
  suitePath: 'CI/k6',
  runStage: 'ci',
  prometheusMetrics: true,
};

// Build the script args for an Argo workflow container
local script = bench.buildScript(suite.serviceUrl, suite);
```

The `buildScript` function returns an array of CLI arguments ready to use as a container command in an Argo workflow template. Fields not set on the suite use the defaults from the generated `Suite` object.

New versions of the library are generated automatically on release via `make libsonnet`. To regenerate locally:

```sh
make libsonnet
```
