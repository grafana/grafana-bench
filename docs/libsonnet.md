# Libsonnet Library

`grafana-bench` offers a [libsonnet library](https://github.com/grafana/deployment_tools/blob/master/ksonnet/lib/argo-workflows-util/common-steps/bench.libsonnet) that facilitates the execution of a bench test suite in ArgoCD workflows.

This library is under regular development and while we don't often deprecate, new things are added regularly. Refer to [latest version](https://github.com/grafana/deployment_tools/blob/master/ksonnet/lib/argo-workflows-util/common-steps/bench.libsonnet) of the library.

You can see the RRC implementation [here](https://github.com/grafana/deployment_tools/blob/master/ksonnet/environments/hosted-grafana-cd/rrc-bench-suites.libsonnet).

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

The following code snipped shows how to use the bench jsonnet library to execute a test suite from the [grafana-api-tests](https://github.com/grafana/grafana-api-tests) test library against multiple grafana instances.

```js
local aw = import 'argo-workflows-util/main.libsonnet';
local steps = aw.group.steps;

runBenchSuite(grafana_url): [
    local suite = {
        benchRevision: 'v0.6.6',
        testType: 'smoke',
        path: 'tests/playlists',
        testRepo: 'https://github.com/grafana/grafana-api-tests.git',
        testRevision: 'v0.3.0',
    };

    aw.testingSteps.benchTest()
    // set environment variables for bench tests from environment secrets
        .withEnvVars([
            envVar.fromSecretRef('GRAFANA_ADMIN_USER', 'grafana-credentials-unified-storage', 'user'),
            envVar.fromSecretRef('GRAFANA_ADMIN_PASSWORD', 'grafana-credentials-unified-storage', 'password'),
            envVar.fromSecretRef('TEST_SUITE_REPO_TOKEN', 'grafana-api-repo-token', 'token'),
        ])
        .withBenchTest(grafana_url, suite)
        .buildStep()
]
```
