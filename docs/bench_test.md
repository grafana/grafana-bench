## bench test

bench test runner

### Synopsis

test subcommand is a wrapper for running a suite of k6 or playwrights tests against a grafana
instance.

The tests to be executed are defined by the --test-suite option.

Tests are parameterized via environment variables following grafana-api-tests
conventions[1].

Supports two kinds of test executions defined by the --test-type option:

* smoke: execute tests and reports failures (default)
* load: execute tests and report execution stats to GCK6

For load tests, if the --k6-cloud-output flag is true, the test results will be
sent to Grafana Cloud k6. The GCK6 credentials[2] must be provided as
environment variables or as arguments (--k6-cloud-project and --k6-cloud-token)

[1] https://github.com/grafana/grafana-api-tests/blob/main/README.md#common-environment-variables
[2] https://grafana.com/docs/grafana-cloud/k6/author-run/tokens-and-cli-authentication/

```
bench test --test-suite /path/to/test/suite [flags]
```

### Examples

```

    bench test --test-suite /path/to/test/folder
    bench test --test-type load --test-suite /path/to/test.js"

```

### Playwright

You can execute playwright by passing the `--runner playwright` options to bench, as per below.

Playwright also requires prepare and execute cmd strings to be passed. On the Bench Docker image node, npm and yarn are available. Playwright browsers are not required to be installed as outlined in detail below.

Bench will overwrite the reporters set in the `playwright.config.ts` via the command line and use the json report to report on the tests.

```

    bench test
        --test-suite /home/bench/work/grafana-plugin-tests/
        --runner playwright
        --pw-prepare-cmd "yarn install"
        --pw-execute-cmd "yarn test"
        --grafana-url "http://host.docker.internal:3000"

```

Currently, there is no way to set the baseURL or executablePath of playwright via the [command line](https://playwright.dev/docs/test-cli). Instead, Bench will pass these values via Environment variable that will need to be referenced in the `playwright.config.ts` file of the project being tested.

`process.env.PLAYWRIGHT_BASE_URL` will be the same value passed as `--grafana-url` in the cli arguments.

`process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH` is set in the docker image of bench. It refers to the chromium executable on the image itself, currently `/usr/bin/chromium`. This is used because the `playwright install` command provided by playwright does not support alpine / musl.

```ts
// Include this into your playwright config
export default defineConfig({
  testDir: "./tests",
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL,
    launchOptions: {
      executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
    },
  },
});
```

### Options

```
      --bench-revision string             grafana bench revision. If not set BENCH_REVISION environment variable is used.
      --dashboard string                  Template for the smoke test suite execution dashboard URL.
                                          Supports the substitution of the following variables:
                                              SuiteRun: identifier of the suite run
                                          Example: http://localhost/dashboards?run={{.SuiteRun}}
      --grafana-password string           grafana password. Overridden by the GRAFANA_PASSWORD environment variable (default "admin")
      --grafana-timeout duration          timeout for waiting grafana to be live (default 1m0s)
      --grafana-url string                url to grafana instance. Overridden by the GRAFANA_URL environment variable (default "http://localhost:3000")
      --grafana-username string           grafana user name. Overridden by the GRAFANA_USER environment variable (default "admin")
  -h, --help                              help for test
      --k6-cloud-output                   send output to GCK6. Requires setting the GCK6 project ID and access token.
      --k6-cloud-project string           K6 cloud project ID. If not set K6_CLOUD_PROJECT_ID environment variable is used
      --k6-cloud-token string             K6 cloud access token. If not set K6_CLOUD_TOKEN environment variable is used
      --k6-verbose                        show k6 test outputs
	  --pw-prepare-cmd                    command used to install dependencies for the test suite eg: npm install
	  --pw-execute-cmd                    command used to execute the test suite eg: npm run test
      --machine-spec string               grafana instance machine spec
      --runner                            test run used to execute the tests defaults to k6, also accepts'playwright'
      --test-report-format string         format of the test execution report. Allowed values 'log' or 'text'.
                                           'log' produced a structure log. 'text' produced an human readable output (default "text")
      --test-suite string                 path to the tests to be executed.
                                          The path must be relative to the base dir (which defaults to the current directory).
                                          A single .js file or a directory can be specified.
                                          If a directory is specified, all .js files in the directory and its sub-directories will be executed.
      --test-suite-base string            base directory for searching test suites. Defaults to current directory
                                          If specified, it is prefixed to the --test-suite.
      --test-suite-name string            test suite name. If not specified, TEST_SUITE_NAME environment variable is used.
                                          Defaults to the last component of --test-suite.
                                          For example --test-suite /path/to/testsuite will give a test suite name of 'testsuite'.
      --test-suite-revision string        test suite revision. If not set TEST_SUITE_REVISION environment variable is used (default "devel")
      --test-suite-revision-file string   path to a file with the test suite revision. Has precedence over test-suite-revision
      --test-trigger string               test trigger (default "local")
      --test-type string                  test type. Allowed values: 'smoke', 'load' (default "smoke")
```

### Options inherited from parent commands

```
      --env string         path to a file with the environment variables.
                           If none is specified and a .env files exists in the work directory, it will be used
      --log-level string   set the log level ('ERROR', 'WARN', 'INFO', 'DEBUG').
                            overridden by the BENCH_LOG_LEVEL environment variable (default "ERROR")
```

### SEE ALSO

- [bench](bench.md) 	- grafana bench

###### Auto generated by spf13/cobra on 30-Apr-2024
