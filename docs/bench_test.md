## bench test

bench test runner

### Synopsis


test subcommand is a wrapper for running a suite of k6 or playwright tests
against a grafana instance.

The tests to be executed are defined by the --test-suite option.

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


[1] https://github.com/grafana/grafana-bench/blob/main/docs/writing_pw_tests.md

Slack Notifications
-------------------
If the --slack-notifications flag is set, test suite failures will be notified using slack.
Notification will be send to the codeowners of the test. Use the --notify-passing option to
send notifications also for passing test suites. 

The --codeowners-mapping argument is used to find the mapping between codeowners and slack channels.

The --slack-token argument provides the slack token. If not provided, the SLACK_TOKEN 
environment variable wil be used. This token requires channel.read, groups.read and chat.write scopes.


```
bench test [flags]
```

### Examples

```

# run a k6 smoke test from the test suite directory
bench test --test-suite /path/to/test/folder

# run a k6 load test using a single test
bench test --test-type load --test-suite /path/to/test.js"

# checkout a test from a repo and run tests from my-branch branch
bench test \
  --test-suite-repo https://url/to/test-repo.git \
  --test-suite-base path/to/local/repo/directory \
  --test-suite-revision my-branch \
  --test-suite tests

# run k6 test with cloud output
bench test \
  --grafana-url "http://host.docker.internal:3000" \
  --test-suite /home/bench/work/grafana-plugin-tests \
  --test-runner k6
  --k6-cloud-output=true

# run k6 test with custom environment variables
bench test \
  --test-suite /home/bench/work/grafana-plugin-tests \
  --test-env-vars VAR=value,ANOTHER_VAR=value        \
  --test-runner k6

# run playwright test
bench test  \
  --grafana-url "http://host.docker.internal:3000" \
  --test-suite grafana-plugin-tests \
  --test-runner playwright \
  --pw-prepare-cmd "yarn install" \
  --pw-execute-cmd "yarn test" \

```

### Options

```
      --bench-revision string           grafana bench revision. If not set BENCH_REVISION environment variable is used.
      --codeowners-mapping string       path or url to the codeowner to slack channel id mapping.
                                        Relative to test suite base dir. (default "codeowners-mapping.yaml")
      --dashboard string                Template for the smoke test suite execution dashboard URL.
                                        Supports the substitution of the following variables:
                                            SuiteRun: identifier of the suite run
                                        Example: http://localhost/dashboards?run={{.SuiteRun}}
      --grafana-admin-password string   grafana admin user's password. Overridden by the GRAFANA_ADMIN_PASSWORD environment variable (default "admin")
      --grafana-admin-user string       grafana admin user name. Overridden by the GRAFANA_ADMIN_USER environment variable (default "admin")
      --grafana-timeout duration        timeout for waiting grafana to be live (default 1m0s)
      --grafana-url string              url to grafana instance. Overridden by the GRAFANA_URL environment variable (default "http://localhost:3000")
  -h, --help                            help for test
      --k6-cloud-output                 send output to GCK6. Requires setting the GCK6 project ID and access token.
      --k6-cloud-project string         K6 cloud project ID. If not set K6_CLOUD_PROJECT_ID environment variable is used
      --k6-cloud-token string           K6 cloud access token. If not set K6_CLOUD_TOKEN environment variable is used
      --notify-passing                  send notifications for passing test suites. By default only not passing test suites are notified
      --pw-execute-cmd string           command used to execute the test suite eg: "npm run test"
      --pw-prepare-cmd string           commands used to install dependencies for the test suite eg: "npm install".
                                        Multiple commands can be specified by separating with ';'.
      --slack-notifications             send notifications to slack. Requires setting the --slack-token option or the SLACK_TOKEN environment variable.
      --slack-token string              slack token used for sending notifications. If not defined SLACK_TOKEN environment variable is used.
                                        The token requires chat:write and channels:read scopes
      --test-env-vars stringToString    custom test environment variables (default [])
      --test-report-format string       format of the test execution report. Allowed values 'log' or 'text'.
                                         'log' produced a structure log. 'text' produced an human readable output (default "text")
      --test-runner string              test runner. Allowed values: 'k6', 'playwright' (default "k6")
      --test-suite string               path to the tests to be executed.
                                        The path must be relative to the base dir (which defaults to the current directory).
                                        A single .js file or a directory can be specified.
                                        If a directory is specified, all .js files in the directory and its sub-directories will be executed.
      --test-suite-base string          base directory for searching test suites. Defaults to current directory
                                        If specified, it is prefixed to the --test-suite.
      --test-suite-name string          test suite name. If not specified, TEST_SUITE_NAME environment variable is used.
                                        Defaults to the last component of --test-suite.
                                        For example --test-suite /path/to/testsuite will give a test suite name of 'testsuite'.
      --test-suite-repo string          repository to get the test suite from. If not set TEST_SUITE_REPO environment variable is used.
                                        If specified, the repo will be checkout into the test-suite-base directory.
                                        If test-suite-revision is specified, that revision will be checkout. Otherwise the default branch will be checkout
      --test-suite-repo-dirs strings    Directories to checkout from test suite repo. If omitted, all folders will be checkout
      --test-suite-repo-token string    authentication token for the test suite repository. If not set TEST_SUITE_REPO_TOKEN environment variable is used.
      --test-suite-revision string      test suite revision. If not set TEST_SUITE_REVISION environment variable is used
      --test-trigger string             test trigger (default "local")
      --test-type string                test type. Allowed values: 'smoke', 'load' (default "smoke")
      --verbose                         show test outputs
```

### Options inherited from parent commands

```
      --env string         path to a file with the environment variables.
                           If none is specified and a .env files exists in the work directory, it will be used
      --log-level string   set the log level ('ERROR', 'WARN', 'INFO', 'DEBUG').
                            overridden by the BENCH_LOG_LEVEL environment variable (default "ERROR")
```

### SEE ALSO

* [bench](bench.md)	 - grafana bench

###### Auto generated by spf13/cobra on 7-Nov-2024
