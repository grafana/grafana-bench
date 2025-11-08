local argo_workflows = (import 'deps/vendor/argo-workflows-libsonnet/main.libsonnet').workflow.v1alpha1;
local base = (import 'deps/_base.libsonnet');
local templates = (import 'deps/utils/templates.libsonnet');
local url = import 'deps/vendor/github.com/jsonnet-libs/xtd/url.libsonnet';
local benchImage = 'us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench';
local benchPlaywrightImage = 'us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench-playwright';

// Generated from bench v1.0.0-test CLI flags
function(name) base(name, templates.bench_test.name) {
  local this = self,
  envVars:: [],

  parameters:: {
    container_patch: std.manifestJsonMinified({
      containers: [{
        name: 'main',
        env: this.envVars,
      }],
    }),
  },

  withEnvVars(envVars):: self {
    envVars+: envVars,
  },

  withContainerImage(image):: self {
    image: image,
  },

  withBenchTest(grafanaURL, suite):: self {
    // Default suite options derived from CLI flags
    local bench_options = {
      // bench image revision - pinned to this version
      benchRevision: 'v1.0.0-test',
      // --go-args: arguments to be passed to go test command (e.g '-tag slow -race')
      goArgs: [],
      // --go-retries: number of retries for failed tests. Retried tests that pass are reported as flaky
      goRetries: 0,
      // --go-test-args: arguments to be passed to the test using the arg flag (e.g '-args -slow 1')
      goTestArgs: [],
      // --go-test-packages: patterns for selecting packages for testing. Can be repeated to specify multiple packages. If no pattern is specified only tests under the current working directory are executed.
      goTestPackages: [],
      // --grafana-admin-password: grafana admin user's password. Overridden by the GRAFANA_ADMIN_PASSWORD environment variable
      grafanaAdminPassword: 'admin',
      // --grafana-admin-user: grafana admin user name. Overridden by the GRAFANA_ADMIN_USER environment variable
      grafanaAdminUser: 'admin',
      // --grafana-timeout: timeout for waiting grafana to be live
      grafanaTimeout: '1m0s',
      // --grafana-url: url to grafana instance. Overridden by the GRAFANA_URL environment variable (default http://localhost:3000)
      grafanaUrl: 'http://localhost:3000',
      // --grafana-version: grafana version. If not provided GRAFANA_VERSION env var is used. If not set, the version is retrieved from the grafana instance.
      grafanaVersion: '',
      // --k6-cloud-output: send output to GCK6. Requires setting the GCK6 project ID and access token.
      k6CloudOutput: false,
      // --k6-cloud-project: K6 cloud project ID. If not set K6_CLOUD_PROJECT_ID environment variable is used
      k6CloudProject: '',
      // --k6-cloud-token: K6 cloud access token. If not set K6_CLOUD_TOKEN environment variable is used
      k6CloudToken: '',
      // --prometheus-metrics: send test suite run results to a prometheus remote write endpoint.
      prometheusMetrics: false,
      // --prometheus-password: prometheus remote write password. If not set PROMETHEUS_PASSWORD environment variable is used.
      prometheusPassword: '',
      // --prometheus-strict-lint: strict lint prometheus metrics. If set to true, will fail if metric does not pass linting
      prometheusStrictLint: false,
      // --prometheus-timeout: prometheus remote write timeout. If not set PROMETHEUS_TIMEOUT environment variable is used.
      prometheusTimeout: '0s',
      // --prometheus-url: prometheus remote write URL. If not set PROMETHEUS_URL environment variable is used.
      prometheusUrl: '',
      // --prometheus-user: prometheus remote write user. If not set PROMETHEUS_USER environment variable is used.
      prometheusUser: '',
      // --pw-execute: command used to execute the test suite eg: "npm run test"
      pwExecute: '',
      // --pw-prepare: commands used to install dependencies for the test suite eg: "npm install". Multiple commands can be specified by separating with ';'.
      pwPrepare: '',
      // --report-output: format of the test execution report. Allowed values 'log' or 'text'. 'log' produced a structure log. 'text' produced an human readable output
      reportOutput: 'text',
      // --run-attribute: adds custom attributes to a suite run. Good for descriptive information. Format: --run-attribute="key=value,key=value". Attributes with no value will be skipped. You can either use the comma separated format shown here or call --run-attribute multiple times to add additional attributes
      runAttribute: [],
      // --run-dashboard: Template for the suite run dashboard URL. Supports the substitution of the following variables: Id: identifier of the suite run Example: http://localhost/dashboards?run={{.Id}}
      runDashboard: '',
      // --run-metric: test suite run custom metrics. Format: name{label=label-value,..}=value. The value must be a valid float number.
      runMetric: [],
      // --run-metrics-file: path to a file containing a list of metrics to be added to the suite run. The file must follow prometheus exposition format. [1] Each non commented line should follow the pattern metric{label1=value1,label2=value2,...} value. The timestamp, if present, is omitted and all metrics are reported using the suite run's execution time. [1] https://github.com/Showmax/prometheus-docs/blob/master/content/docs/instrumenting/exposition_formats.md
      runMetricsFile: '',
      // --run-metrics-prefix: prefix to append to the suite run metric names
      runMetricsPrefix: '',
      // --run-stage: the stage of CI the suite was executed. For example, 'local', 'ci', 'rrc'.
      runStage: 'local',
      // --slack-codeowners-mapping: path or url to the codeowner to slack channel id mapping. Relative to test suite base dir.
      slackCodeownersMapping: 'codeowners-mapping.yaml',
      // --slack-notifications: send notifications to slack. Requires setting the --slack-token option or the SLACK_TOKEN environment variable.
      slackNotifications: false,
      // --slack-passing: send notifications for passing test suites. By default only not passing test suites are notified
      slackPassing: false,
      // --slack-token: slack token used for sending notifications. If not defined SLACK_TOKEN environment variable is used. The token requires chat:write and channels:read scopes
      slackToken: '',
      // --suite-base: base directory for searching test suites. Defaults to current directory If specified, it is prefixed to the --suite-path.
      suiteBase: '',
      // --suite-name: test suite name. If not specified, SUITE_NAME environment variable is used. Defaults to the last component of -suite-path. For example --suite--path path/to/testsuite will give a test suite name of 'testsuite'.
      suiteName: '',
      // --suite-path: path to the tests to be executed. The path must be relative to the base dir (which defaults to the current directory). A single .js file or a directory can be specified. If a directory is specified, all files in the directory and its sub-directories will be executed.
      suitePath: error 'must define suite path',
      // --suite-repo-dirs: Directories to checkout from test suite repo. If omitted, all folders will be checkout
      suiteRepoDirs: [],
      // --suite-repo-token: authentication token for the test suite repository. If not set SUITE_REPO_TOKEN environment variable is used.
      suiteRepoToken: '',
      // --suite-repo-url: url to the repository to get the test suite from. If not set SUITE_REPO_URL environment variable is used. If specified, the repo will be checkout into the --suite-base directory. If --suite-revision is specified, that revision will be checkout. Otherwise the default branch will be checkout
      suiteRepoUrl: '',
      // --suite-revision: test suite revision. If not set SUITE_REVISION environment variable is used
      suiteRevision: '',
      // --test-env: environment variables passed to the test execution.
      testEnv: [],
      // --test-runner: test runner. Allowed values: 'k6', 'playwright', 'go'
      testRunner: 'k6',
      // --test-type: test type. Allowed values: 'smoke', 'load'
      testType: 'smoke',
      // --test-verbose: show test output
      testVerbose: false,
      // suite execution options
      options: {
        // prevent the workflow step to fail if the test suite fails
        noFail: false,
      },
      // environment variables to be passed to the test step in the workflow
      envVars: [],
    } + suite,

    // set step's env vars for test
    envVars+: bench_options.envVars,

    // create list of env=$env
    local test_env_vars = [env + '="$"' + env for env in bench_options.testEnv],

    local script = [
                     'grafana-bench',
                     'test',
                     '--grafana-url',
                     grafanaURL,
                     '--suite-path',
                     bench_options.path,
                     '--log-level',
                     'info',
                     '--test-report-format',
                     'log',
                   ] + std.flattenArrays([['--go-args', item] for item in bench_options.goArgs])
                   + (if bench_options.goRetries != 0 then ['--go-retries', std.toString(bench_options.goRetries)] else [])
                   + std.flattenArrays([['--go-test-args', item] for item in bench_options.goTestArgs])
                   + std.flattenArrays([['--go-test-packages', item] for item in bench_options.goTestPackages])
                   + (if bench_options.grafanaAdminPassword != '' then ['--grafana-admin-password', bench_options.grafanaAdminPassword] else [])
                   + (if bench_options.grafanaAdminUser != '' then ['--grafana-admin-user', bench_options.grafanaAdminUser] else [])
                   + (if bench_options.grafanaTimeout != '0s' then ['--grafana-timeout', bench_options.grafanaTimeout] else [])
                   + (if bench_options.grafanaVersion != '' then ['--grafana-version', bench_options.grafanaVersion] else [])
                   + (if bench_options.k6CloudOutput then ['--k6-cloud-output'] else [])
                   + (if bench_options.k6CloudProject != '' then ['--k6-cloud-project', bench_options.k6CloudProject] else [])
                   + (if bench_options.k6CloudToken != '' then ['--k6-cloud-token', bench_options.k6CloudToken] else [])
                   + (if bench_options.prometheusMetrics then ['--prometheus-metrics'] else [])
                   + (if bench_options.prometheusPassword != '' then ['--prometheus-password', bench_options.prometheusPassword] else [])
                   + (if bench_options.prometheusStrictLint then ['--prometheus-strict-lint'] else [])
                   + (if bench_options.prometheusTimeout != '0s' then ['--prometheus-timeout', bench_options.prometheusTimeout] else [])
                   + (if bench_options.prometheusUrl != '' then ['--prometheus-url', bench_options.prometheusUrl] else [])
                   + (if bench_options.prometheusUser != '' then ['--prometheus-user', bench_options.prometheusUser] else [])
                   + (if bench_options.pwExecute != '' then ['--pw-execute', std.escapeStringBash(bench_options.pwExecute)] else [])
                   + (if bench_options.pwPrepare != '' then ['--pw-prepare', std.escapeStringBash(bench_options.pwPrepare)] else [])
                   + (if bench_options.reportOutput != '' then ['--report-output', bench_options.reportOutput] else [])
                   + std.flattenArrays([['--run-attribute', item] for item in bench_options.runAttribute])
                   + (if bench_options.runDashboard != '' then ['--run-dashboard', url.escapeString(bench_options.runDashboard)] else [])
                   + std.flattenArrays([['--run-metric', item] for item in bench_options.runMetric])
                   + (if bench_options.runMetricsFile != '' then ['--run-metrics-file', bench_options.runMetricsFile] else [])
                   + (if bench_options.runMetricsPrefix != '' then ['--run-metrics-prefix', bench_options.runMetricsPrefix] else [])
                   + (if bench_options.runStage != '' then ['--run-stage', bench_options.runStage] else [])
                   + (if bench_options.slackCodeownersMapping != '' then ['--slack-codeowners-mapping', bench_options.slackCodeownersMapping] else [])
                   + (if bench_options.slackNotifications then ['--slack-notifications'] else [])
                   + (if bench_options.slackPassing then ['--slack-passing'] else [])
                   + (if bench_options.slackToken != '' then ['--slack-token', bench_options.slackToken] else [])
                   + (if bench_options.suiteBase != '' then ['--suite-base', bench_options.suiteBase] else [])
                   + (if bench_options.suiteName != '' then ['--suite-name', bench_options.suiteName] else [])
                   + std.flattenArrays([['--suite-repo-dirs', dir] for dir in bench_options.suiteRepoDirs])
                   + (if bench_options.suiteRepoToken != '' then ['--suite-repo-token', bench_options.suiteRepoToken] else [])
                   + (if bench_options.suiteRepoUrl != '' then ['--suite-repo-url', bench_options.suiteRepoUrl] else [])
                   + (if bench_options.suiteRevision != '' then ['--suite-revision', bench_options.suiteRevision] else [])
                   + (if test_env_vars != [] then ['--test-env', std.join(',', test_env_vars)] else [])
                   + (if bench_options.testRunner != '' then ['--test-runner', bench_options.testRunner] else [])
                   + (if bench_options.testType != '' then ['--test-type', bench_options.testType] else [])
                   + (if bench_options.testVerbose then ['--test-verbose'] else [])
                   + ([if bench_options.options.noFail then '|| true']),  // MUST be the last option

    // Helper function to select the appropriate image based on test runner
    local selectedImage =
      if bench_options.testRunner == 'playwright' then
        benchPlaywrightImage
      else
        benchImage,

    parameters+: {
      script: std.join(' ', script),
      image: selectedImage + ':' + bench_options.benchRevision,
    },
  },
}
