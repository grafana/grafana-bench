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

Execute go test suites. The '--suite-path' is used as a pattern for selecting the
packages to test. It must start with './' to test local packages. To ensure sub-
packages are also included, add '/...' at the end. For example:

    grafana-bench test --test-runner go --suite-path ./path/to/package/...

Additional arguments such as build tags can be passed using the --go-args flag.
    grafana-bench test --test-runner go \
       --go-args "-tags=slow -race -timeout=30m" \
       --suite-path ./path/to/package/...


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
bench test --suite-path /path/to/test/folder

# run a k6 load test using a single test
bench test --test-type load --suite-path /path/to/test.js"

# checkout a test from a repo and run tests from my-branch branch
bench test \
  --suite-repo-url https://url/to/test-repo.git \
  --suite-base path/to/local/repo/directory \
  --suite-revision my-branch \
  --suite-path tests

# run k6 test with cloud output
bench test \
  --grafana-url "http://host.docker.internal:3000" \
  --suite-path /home/bench/work/grafana-plugin-tests \
  --test-runner k6
  --k6-cloud-output=true

# run k6 test with custom environment variables
bench test \
  --suite-path /home/bench/work/grafana-plugin-tests \
  --test-env VAR=value,ANOTHER_VAR=value        \
  --test-runner k6

# run playwright test
bench test  \
  --grafana-url "http://host.docker.internal:3000" \
  --suite-path grafana-plugin-tests \
  --test-runner playwright \
  --pw-prepare "yarn install" \
  --pw-execute "yarn test" \

# run go test
bench test  \
  --suite-path ./path/to/test/... \
  --test-runner go

```

### Options

```
      --bench-revision string             grafana bench revision. If not set BENCH_REVISION environment variable is used.
                                          If not set, the current git revision is used (default (devel)  (default "(devel)")
      --codeowners-mapping string         deprecated. Use slack-codeowners-mapping (default "codeowners-mapping.yaml")
      --dashboard string                  deprecated. Use run-dashboard
      --format string                     deprecated. Use report-output
      --grafana-admin-password string     grafana admin user's password. Overridden by the GRAFANA_ADMIN_PASSWORD environment variable (default "admin")
      --grafana-admin-user string         grafana admin user name. Overridden by the GRAFANA_ADMIN_USER environment variable (default "admin")
      --grafana-timeout duration          timeout for waiting grafana to be live (default 1m0s)
      --grafana-url string                url to grafana instance. Overridden by the GRAFANA_URL environment variable (default http://localhost:3000) (default "http://localhost:3000")
      --grafana-version string            grafana version. If not provided GRAFANA_VERSION env var is used.
                                          If not set, the version is retrieved from the grafana instance.
  -h, --help                              help for test
      --k6-cloud-output                   send output to GCK6. Requires setting the GCK6 project ID and access token.
      --k6-cloud-project string           K6 cloud project ID. If not set K6_CLOUD_PROJECT_ID environment variable is used
      --k6-cloud-project-id string        deprecated. Use k6-cloud-project
      --k6-cloud-token string             K6 cloud access token. If not set K6_CLOUD_TOKEN environment variable is used
      --notify-passing                    deprecated. Use slack-notify-passing
      --prometheus-metrics                send test suite run results to a prometheus remote write endpoint.
      --prometheus-password string        prometheus remote write password. If not set PROMETHEUS_PASSWORD environment variable is used.
      --prometheus-strict-lint            strict lint prometheus metrics. If set to true, will fail if metric does not pass linting
      --prometheus-timeout duration       prometheus remote write timeout. If not set PROMETHEUS_TIMEOUT environment variable is used.
      --prometheus-url string             prometheus remote write URL. If not set PROMETHEUS_URL environment variable is used.
      --prometheus-user string            prometheus remote write user. If not set PROMETHEUS_USER environment variable is used.
      --pw-execute string                 command used to execute the test suite eg: "npm run test"
      --pw-execute-cmd string             deprecated. Use pw-execute
      --pw-prepare string                 commands used to install dependencies for the test suite eg: "npm install".
                                          Multiple commands can be specified by separating with ';'.
      --pw-prepare-cmd string             deprecated. Use pw-prepare
      --report-format string              deprecated. Use report-output (default "text")
      --report-output string              format of the test execution report. Allowed values 'log' or 'text'.
                                           'log' produced a structure log. 'text' produced an human readable output (default "text")
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
      --run-trigger string                trigger of bench execution. For example, 'ci' or 'local'. (default "local")
      --slack-codeowners-mapping string   path or url to the codeowner to slack channel id mapping.
                                          Relative to test suite base dir. (default "codeowners-mapping.yaml")
      --slack-notifications               send notifications to slack. Requires setting the --slack-token option or the SLACK_TOKEN environment variable.
      --slack-passing                     send notifications for passing test suites. By default only not passing test suites are notified
      --slack-token string                slack token used for sending notifications. If not defined SLACK_TOKEN environment variable is used.
                                          The token requires chat:write and channels:read scopes
      --suite-base string                 base directory for searching test suites. Defaults to current directory
                                          If specified, it is prefixed to the --suite-path.
      --suite-name string                 test suite name. If not specified, SUITE_NAME environment variable is used.
                                          Defaults to the last component of -suite-path.
                                          For example --suite--path path/to/testsuite will give a test suite name of 'testsuite'.
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
      --suite-run-metrics strings         deprecated use --run-metrics
      --suite-run-metrics-prefix string   deprecated. Use --run-metrics-prefix
      --test-env stringToString           environment variables passed to the test execution. (default [])
      --test-env-vars stringToString      deprecated. Use test-env (default [])
      --test-report-format string         deprecated. Use report-output
      --test-runner string                test runner. Allowed values: 'k6', 'playwright', 'go' (default "k6")
      --test-suite string                 deprecated. Use suite-path
      --test-suite-base string            deprecated. Use suite-base
      --test-suite-name string            deprecated. Use suite-name
      --test-suite-repo string            deprecated. Use suite-repo-url
      --test-suite-repo-dirs strings      deprecated. Use suite-repo-dirs
      --test-suite-repo-token string      deprecated. Use suite-repo-token
      --test-suite-revision string        deprecated. Use suite-revision
      --test-trigger string               deprecated. Use run-trigger (default "local")
      --test-type string                  test type. Allowed values: 'smoke', 'load' (default "smoke")
      --trigger string                    deprecated. Use run-trigger (default "local")
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

###### Auto generated by spf13/cobra on 30-Apr-2025
