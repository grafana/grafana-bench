local bench = import 'test-output/bench.libsonnet';

// Create an Argo workflow that uses the bench libsonnet
{
  apiVersion: 'argoproj.io/v1alpha1',
  kind: 'Workflow',
  metadata: {
    name: 'bench-smoke-test',
    namespace: 'default',
  },
  spec: {
    entrypoint: 'bench-step',
    templates: [
      bench('bench-step').withBenchTest('http://localhost:3000', {
        testRunner: 'k6',
        path: 'CI/k6',
        type: 'smoke',
      })
    ]
  }
}
