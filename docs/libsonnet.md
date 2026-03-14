# Libsonnet Library

`grafana-bench` offers a libsonnet library that facilitates the execution of a bench test suite in ArgoCD workflows.

## Suite Definition

A test suite is defined by the following Jsonnet object:

```js
{
      // bench image revision. Required
      benchRevision: error 'must define the bench version',
      // name of the test suite
      name: '',
      // test trigger (e.g. rcc for rolling release channels)
      trigger: '',
      // type of test 'smoke' or 'load'
      testType: 'smoke',
      // url to the repository to fetch the test from
      testRepo: '',
      // test revision. If omitted the main branch of the repository will be used
      testRevision: '',
      // path to the tests with respect of the root of the test repository. Required
      path: error 'must define path to tests',
      // List of environment variables to be passed to the test step in the workflow
      // Used to override bench options. E.g. GRAFANA_URL
      envVars: [],
      // list of variables to be passed to the test
      testEnvVars: [],
      // generate test output
      verbose: false,
      // suite execution options
      options: {
        // prevent the workflow step to fail if the test suite fails
        noFail: false,
      },
}
```

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
