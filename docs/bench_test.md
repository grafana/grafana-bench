## bench test

bench test runner

### Synopsis


test subcommand is a wrapper for running a suite of tests
against a grafana instance.

The tests to be executed are defined by the --suite-path option.

The --test-runner option defines the type of test to execute. The default is k6.

Supports two kinds of test executions defined by the --test-type option:
* smoke: execute tests and reports failures (default)
* load: execute tests and report execution stats

Exit Codes
----------
The test command uses different exit codes to indicate the reason for failure:
* 0: Success - all tests passed
* 1: Test failure - one or more tests failed
* 2: Internal error - configuration, execution, or system error

This allows CI systems and automation tools to distinguish between test
failures (which may be expected during development) and internal errors
(which typically require immediate attention).

k6
--
Executes a test suite using k6.

For load tests, if the --k6-cloud-output flag is true, the test results will be
sent to Grafana Cloud k6. The GCK6 credentials[1] must be provided as
environment variables or as arguments (--k6-cloud-project and --k6-cloud-token)

Tests are parameterized via environment variables following grafana-api-tests
conventions[2].

[1] https://grafana.com/docs/grafana-cloud/k6/author-run/tokens-and-cli-authentication/
[2] https://github.com/grafana/grafana-api-tests/blob/main/README.md#common-environment-variables


Playwright
----------

Executes a test suite using playwright.

The --pw-prepare and --pw-execute arguments define the commands to be execute for 
preparing and executing the tests

The url to the grafana instance defined in the --grafana-url cli arguments will 
be passed to the test in the PLAYWRIGHT_BASE_URL environment variable.
See [1] for details on how to develop playwright tests compatible with the bench
test runner.

Go
--

Execute go a test suite. It wraps go test command and reports the results using the bench reporters.
It also provides the possibility of retrying failed test and report flaky tests.

The --go-test-packages argument defines the packages to be tested using the same format as the go test command.
If not specified, only the tests under the current working directory are executed.
    grafana-bench test \
      --suite-name my-repo/go-tests \
      --test-runner go \
      --go-test-packages ./path/to/packages/...

The '--suite-path' can be used to change the working directory for the go test command.
The following command will search for tests under the 'tests' directory using the pattern defined by
the --go-test-packages:
    grafana-bench test \
      --suite-name my-repo/go-tests \
      --test-runner go \
      --suite-path tests \
      --go-test-packages ./path/to/package/...

Additional arguments such as build tags can be passed using the --go-args flag.
    grafana-bench test \
       --suite-name my-repo/go-tests \
       --test-runner go \
       --go-test-packages ./path/to/package/... \
       --go-args "-tags=slow -race -timeout=30m"

For passing flags to configure the test, use the --go-test-args flag
    grafana-bench test \
       --suite-name my-repo/go-tests \
       --test-runner go \
       --go-test-packages ./path/to/package/... \
       --go-test-args "-slow 1"

The go test executor can retry failed tests. Test that pass after retrying are reported as flaky.
The number of retries is defined by the go-retries option.
    grafana-bench test \
       --suite-name my-repo/go-tests \
       --test-runner go \
       --go-test-packages ./path/to/package/... \
       --go-retries 3

Go Benchmarks
-------------

Execute Go benchmarks with 'go test -bench' and export performance metrics (ns/op, B/op, allocs/op) to
Prometheus and test reports. Use --test-runner gobench to run benchmarks instead of tests.

The --gobench-packages argument defines the packages to benchmark using the same format as go test.
If not specified, defaults to './...' (all packages).
    grafana-bench test \
      --suite-name my-repo/benchmarks \
      --test-runner gobench \
      --gobench-packages ./pkg/...

Use --gobench-pattern to filter benchmarks by name (regex):
    grafana-bench test \
      --suite-name my-repo/benchmarks \
      --test-runner gobench \
      --gobench-pattern "BenchmarkAPI"

Control benchmark duration with --gobench-time (e.g., "10s" or "100x"):
    grafana-bench test \
      --suite-name my-repo/benchmarks \
      --test-runner gobench \
      --gobench-time 10s

Memory statistics are enabled by default. Disable with --gobench-mem=false:
    grafana-bench test \
      --suite-name my-repo/benchmarks \
      --test-runner gobench \
      --gobench-mem=false

Run benchmarks multiple times with --gobench-count:
    grafana-bench test \
      --suite-name my-repo/benchmarks \
      --test-runner gobench \
      --gobench-count 5

Additional go test arguments can be passed using --gobench-args:
    grafana-bench test \
       --suite-name my-repo/benchmarks \
       --test-runner gobench \
       --gobench-args "-tags=integration -timeout=30m"

Arguments for the benchmark itself can be passed using --gobench-bench-args:
    grafana-bench test \
       --suite-name my-repo/benchmarks \
       --test-runner gobench \
       --gobench-bench-args "-cpuprofile=cpu.prof"

Complete example with Prometheus metrics:
    grafana-bench test \
       --suite-name my-repo/benchmarks \
       --service bench \
       --service-version abc123 \
       --test-runner gobench \
       --suite-path ./benchmarks \
       --gobench-pattern "BenchmarkAPI" \
       --gobench-time 10s \
       --gobench-count 3 \
       --prometheus-metrics \
       --prometheus-url https://prometheus.example.com/api/v1/write \
       --run-stage ci

[1] https://github.com/grafana/grafana-bench/blob/main/docs/writing_pw_tests.md

Slack Notifications
-------------------
If the --slack-notifications flag is set, test suite failures will be notified using slack.
Use the --slack-passing option to send notifications also for passing test suites. 

Notification will be send to the codeowners of the test. The --codeowners-mapping argument
is used to find the mapping between codeowners and slack channels.

The --slack-token argument provides the slack token. If not provided, the SLACK_TOKEN 
environment variable wil be used. This token requires channel.read, groups.read and chat.write scopes.

Configuration File
------------------

The test command supports reading configuration from a YAML file. The default file is bench.yaml.
The file can be specified using the --config flag.

The configuration file can contain any of the flags supported by the test command.

As a convention, a flag with the name "--foo-bar" in the command line will be
represented in the configuration file as:
   foo:
     bar: value

Notice that some flag names have changed to accommodate the configuration file format.

** NOTE: Deprecated flag names are not supported in the configuration file. **

Precedence: The flags specified on the command line and the environment variables will take precedence over the
values in the configuration file.

** NOTE: we strongly discourage storing sensitive information such as tokens in the configuration file. **
Consider using the environment variables or secrets manager for storing sensitive information.
Also consider using .env files for storing this information in local development environments.

# bench.yaml example

test:
  env:
    VAR1: "value1"
    VAR2: "value2"
  type: "smoke"
  runner: "k6"

report:
  output: "text"

suite:
  name: "my-test-suite"
  path: "/path/to/tests"
  repo: "https://github.com/org/test-repo.git"
  revision: "main"
  
grafana:
  url: "http://localhost:3000"
  admin
    user: "admin"
    password: "secret"

k6:
  cloud:
    output: true
    token: "your-token"
    project: "your-project-id"

pw:
  prepare: "npm install"
  execute: "npm test"

slack:
  notifications: true
  passing: false
  token: "xoxb-your-token"

run:
  trigger: "ci"


```
bench test [flags]
```

### Examples

```

# run a k6 smoke test from the test suite directory
bench test \
  --suite-name my-repo/smoke-tests \
  --suite-path /path/to/test/folder

# run a k6 load test using a single test
bench test \
  --suite-name my-repo/load-test \
  --test-type load \
  --suite-path /path/to/test.js"

# checkout a test from a repo and run tests from my-branch branch
bench test \
  --suite-name my-repo/integration-tests \
  --suite-repo-url https://url/to/test-repo.git \
  --suite-base path/to/local/repo/directory \
  --suite-revision my-branch \
  --suite-path tests

# run k6 test with cloud output
bench test \
  --suite-name my-repo/k6-tests \
  --grafana-url "http://host.docker.internal:3000" \
  --suite-path /home/bench/work/grafana-plugin-tests \
  --test-runner k6
  --k6-cloud-output=true

# run k6 test with custom environment variables
bench test \
  --suite-name my-repo/k6-tests \
  --suite-path /home/bench/work/grafana-plugin-tests \
  --test-env VAR=value,ANOTHER_VAR=value        \
  --test-runner k6

# run playwright test
bench test  \
  --suite-name my-repo/e2e-tests \
  --grafana-url "http://host.docker.internal:3000" \
  --suite-path grafana-plugin-tests \
  --test-runner playwright \
  --pw-prepare "yarn install; playwright install chromium" \
  --pw-execute "yarn test" \

# run go test
bench test  \
  --suite-name my-repo/go-tests \
  --suite-path ./path/to/test/... \
  --test-runner go

# run go benchmarks with prometheus metrics
bench test \
  --suite-name my-repo/benchmarks \
  --service bench \
  --service-version v1.0.0 \
  --test-runner gobench \
  --suite-path ./benchmarks \
  --gobench-pattern "BenchmarkAPI" \
  --gobench-time 10s \
  --prometheus-metrics \
  --run-stage ci

```

### Options

```
      --bench-revision string             grafana bench revision. If not set BENCH_REVISION environment variable is used.
                                          If not set, the current git revision is used (default (devel)  (default "(devel)")
      --fetch-grafana-version string      Optional: Fetch Grafana version from API using provided credentials in 'user:password' format.
                                          Mutually exclusive with --service-version. Overridden by FETCH_GRAFANA_VERSION environment variable.
                                          Example: --fetch-grafana-version=admin:admin
      --git-driver string                 git driver used for downloading the test suite repo ('nanogit', 'gogit'). (default "nanogit")
      --go-args stringArray               arguments to be passed to go test command (e.g '-tag slow -race')
      --go-retries int                    number of retries for failed tests. Retried tests that pass are reported as flaky
      --go-test-args stringArray          arguments to be passed to the test using the arg flag (e.g '-args -slow 1')
      --go-test-packages stringArray      patterns for selecting packages for testing. Can be repeated to specify multiple packages.
                                          If no pattern is specified only tests under the current working directory are executed.
      --gobench-args stringArray          Additional arguments passed to 'go test' (e.g., '-tags=integration')
      --gobench-bench-args stringArray    Arguments passed to benchmarks via -args
      --gobench-count int                 Number of times to run each benchmark (default 1)
      --gobench-mem                       Enable memory statistics (B/op, allocs/op) (default true)
      --gobench-packages stringArray      Package patterns for benchmarks (e.g., './...', './pkg/...'). If not specified, defaults to './...'
      --gobench-pattern string            Benchmark name pattern (regex). Use '.' to run all benchmarks (default ".")
      --gobench-time string               Benchmark duration (e.g., '10s', '100x'). If not set, uses Go's default
  -h, --help                              help for test
      --k6-cloud-output                   send output to GCK6. Requires setting the GCK6 project ID and access token.
      --k6-cloud-project string           K6 cloud project ID. If not set K6_CLOUD_PROJECT_ID environment variable is used
      --k6-cloud-token string             K6 cloud access token. If not set K6_CLOUD_TOKEN environment variable is used
      --prometheus-metrics                send test suite run results to a prometheus remote write endpoint.
      --prometheus-password string        prometheus remote write password. If not set PROMETHEUS_PASSWORD environment variable is used.
      --prometheus-strict-lint            strict lint prometheus metrics. If set to true, will fail if metric does not pass linting
      --prometheus-timeout duration       prometheus remote write timeout. If not set PROMETHEUS_TIMEOUT environment variable is used.
      --prometheus-url string             prometheus remote write URL. If not set PROMETHEUS_URL environment variable is used.
      --prometheus-user string            prometheus remote write user. If not set PROMETHEUS_USER environment variable is used.
      --pw-execute string                 command used to execute the test suite eg: "npm run test"
      --pw-prepare string                 commands used to install dependencies for the test suite eg: "npm install".
                                          Multiple commands can be specified by separating with ';'.
      --report-output string              format of the test execution report. Allowed values 'log' or 'text'.
                                           'log' produced a structure log. 'text' produced an human readable output (default "text")
      --run-attribute stringArray         adds custom attributes to a suite run. Good for descriptive information. Format: --run-attribute="key=value,key=value". Attributes with no value will be skipped. You can either use the comma separated format shown here or call --run-attribute multiple times to add additional attributes
      --run-dashboard string              Template for the suite run dashboard URL.
                                          Supports the substitution of the following variables:
                                              Id: identifier of the suite run
                                          Example: http://localhost/dashboards?run={{.Id}}
      --run-metric stringArray            test suite run custom metrics. Format: name{label=label-value,..}=value. The value must be a valid float number.
      --run-metrics-file string           path to a file containing a list of metrics to be added to the suite run.
                                          The file must follow prometheus exposition format. [1]
                                          Each non commented line should follow the pattern metric{label1=value1,label2=value2,...} value.
                                          The timestamp, if present, is omitted and all metrics are reported using the suite run's execution time.
                                          [1] https://github.com/Showmax/prometheus-docs/blob/master/content/docs/instrumenting/exposition_formats.md
      --run-metrics-prefix string         prefix to append to the suite run metric names
      --run-stage string                  the stage of CI the suite was executed. For example, 'local', 'ci', 'rrc'. (default "local")
      --service string                    REQUIRED. Name of the service being tested (e.g., 'grafana', 'loki', 'tempo', 'datasources'). Used for identifying which service the test results belong to in logs and metrics.
      --service-health-check              Perform a TCP health check on the service before running tests. Uses --service-url and --service-timeout.
      --service-timeout duration          timeout for waiting for the service to be live (default 1m0s)
      --service-url string                URL to the service being tested. Overridden by the SERVICE_URL environment variable (default http://localhost:3000) (default "http://localhost:3000")
      --service-version string            REQUIRED. Version of the service being tested (e.g., '11.0.0', '2.9.0'). Overridden by the SERVICE_VERSION environment variable.
      --slack-codeowners-mapping string   path or url to the codeowner to slack channel id mapping.
                                          Relative to test suite base dir. (default "codeowners-mapping.yaml")
      --slack-notifications               send notifications to slack. Requires setting the --slack-token option or the SLACK_TOKEN environment variable.
      --slack-passing                     send notifications for passing test suites. By default only not passing test suites are notified
      --slack-token string                slack token used for sending notifications. If not defined SLACK_TOKEN environment variable is used.
                                          The token requires chat:write and channels:read scopes
      --suite-base string                 base directory for searching test suites. Defaults to current directory
                                          If specified, it is prefixed to the --suite-path. (default ".")
      --suite-name string                 [REQUIRED] Test suite name used for identifying and labeling test results in logs and metrics.
                                          If not specified, SUITE_NAME environment variable is used.
                                          Example: 'grafana-bench/go-tests' or 'my-repo/smoke-tests'
      --suite-path string                 path to the tests to be executed.
                                          The path must be relative to the base dir (which defaults to the current directory).
                                          A single .js file or a directory can be specified.
                                          If a directory is specified, all files in the directory and its sub-directories will be executed.
      --suite-repo-dirs strings           Directories to checkout from test suite repo. If omitted, all folders will be checkout
      --suite-repo-token string           authentication token for the test suite repository. 
                                          If not set SUITE_REPO_TOKEN environment variable is used.
      --suite-repo-url string             url to the repository to get the test suite from. If not set SUITE_REPO_URL environment variable is used.
                                          If specified, the repo will be checkout into the --suite-base directory.
                                          If --suite-revision is specified, that revision will be checkout.
                                          Otherwise the default branch will be checkout
      --suite-revision string             test suite revision. If not set SUITE_REVISION environment variable is used
      --test-env strings                  environment variables passed to the test execution. Use 'KEY=VALUE' to set explicitly, or 'KEY' to pass through from environment (secure for credentials).
      --test-runner string                test runner. Allowed values: 'k6', 'playwright', 'go', 'gobench' (default "k6")
      --test-type string                  test type. Allowed values: 'smoke', 'load' (default "smoke")
      --test-verbose                      show test output
```

### Options inherited from parent commands

```
      --config string      path to config file (default "bench.yaml")
      --env string         path to a file with the environment variables.
                           If none is specified and a .env files exists in the work directory, it will be used
      --log-level string   set the log level ('ERROR', 'WARN', 'INFO', 'DEBUG').
                            overridden by the BENCH_LOG_LEVEL environment variable (default "ERROR")
```

### SEE ALSO

* [bench](bench.md)	 - grafana bench

###### Auto generated by spf13/cobra on 28-Jan-2026
