## bench report

report test suite execution results

### Synopsis


report subcommand reports test suite execution results.

Presently if supports only playwright test results in json format.

It produces a human readable output or a structured log output based on the format flag.

When using the log format, in order to report the test suite execution results, the following
information is needed:
- trigger
- test type
- suite name
- suite revision (optional)
- bench revision (optional)
- grafana version

If the grafana version is not provided, the reporter will connect to the grafana instance
using the admin user and password provided and get the version.


```
bench report [flags]
```

### Examples

```

grafana-bench report \
  --trigger local \
  --test-type smoke \
  --suite-name smoke-test \
  --report-input playwright \
  --report-output log /path/to/playwright/report.json


Configuration File
------------------

The report command supports reading configuration from a YAML file. The default file is bench.yaml.
The file can be specified using the --config flag.

The configuration file can contain any of the flags supported by the report command.

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
trigger: "ci"

test:
  type: "smoke"

report:
    input: "playwright"
    output: "text"

suite:
  name: "my-test-suite"
  revision: "main"
  
grafana:
  url: "http://localhost:3000"
  version: "v10.0.0"

```

### Options

```
      --bench-revision string            grafana bench revision. If not set BENCH_REVISION environment variable is used.
                                         If not set, the current git revision is used (default (devel)  (default "(devel)")
      --fetch-grafana-version string     Optional: Fetch Grafana version from API using provided credentials in 'user:password' format.
                                         Mutually exclusive with --service-version. Overridden by FETCH_GRAFANA_VERSION environment variable.
                                         Example: --fetch-grafana-version=admin:admin
  -h, --help                             help for report
      --js-coverage-codeowner string     Code owner for JavaScript coverage metrics (e.g., @grafana/datapro)
      --js-coverage-package string       Package name for JavaScript coverage metrics (e.g., @grafana/grafana)
      --prometheus-metrics               send test suite run results to a prometheus remote write endpoint.
      --prometheus-password string       prometheus remote write password. If not set PROMETHEUS_PASSWORD environment variable is used.
      --prometheus-strict-lint           strict lint prometheus metrics. If set to true, will fail if metric does not pass linting
      --prometheus-timeout duration      prometheus remote write timeout. If not set PROMETHEUS_TIMEOUT environment variable is used.
      --prometheus-url string            prometheus remote write URL. If not set PROMETHEUS_URL environment variable is used.
      --prometheus-user string           prometheus remote write user. If not set PROMETHEUS_USER environment variable is used.
      --report-input string              report input format. Valid values are 'playwright', 'go', 'zizmor', and 'trufflehog'
      --report-output string             format of the test execution report. Allowed values 'log' or 'text'.
                                          'log' produced a structure log. 'text' produced an human readable output (default "text")
      --run-attribute stringArray        adds custom attributes to a suite run. Good for descriptive information. Format: --run-attribute="key=value,key=value". Attributes with no value will be skipped. You can either use the comma separated format shown here or call --run-attribute multiple times to add additional attributes
      --run-dashboard string             Template for the suite run dashboard URL.
                                         Supports the substitution of the following variables:
                                             Id: identifier of the suite run
                                         Example: http://localhost/dashboards?run={{.Id}}
      --run-metric stringArray           test suite run custom metrics. Format: name{label=label-value,..}=value. The value must be a valid float number.
      --run-metrics-file string          path to a file containing a list of metrics to be added to the suite run.
                                         The file must follow prometheus exposition format. [1]
                                         Each non commented line should follow the pattern metric{label1=value1,label2=value2,...} value.
                                         The timestamp, if present, is omitted and all metrics are reported using the suite run's execution time.
                                         [1] https://github.com/Showmax/prometheus-docs/blob/master/content/docs/instrumenting/exposition_formats.md
      --run-metrics-prefix string        prefix to append to the suite run metric names
      --run-stage string                 the stage of CI the suite was executed. For example, 'local', 'ci', 'rrc'. (default "local")
      --service string                   REQUIRED. Name of the service being tested (e.g., 'grafana', 'loki', 'tempo', 'datasources'). Used for identifying which service the test results belong to in logs and metrics.
      --service-health-check             Perform a TCP health check on the service before running tests. Uses --service-url and --service-timeout.
      --service-timeout duration         timeout for waiting for the service to be live (default 1m0s)
      --service-url string               URL to the service being tested. Overridden by the SERVICE_URL environment variable (default http://localhost:3000) (default "http://localhost:3000")
      --service-version string           REQUIRED. Version of the service being tested (e.g., '11.0.0', '2.9.0'). Overridden by the SERVICE_VERSION environment variable.
      --suite-name string                [REQUIRED] Test suite name used for identifying and labeling test results in logs and metrics.
                                         If not specified, SUITE_NAME environment variable is used.
                                         Example: 'grafana-bench/go-tests' or 'my-repo/smoke-tests'
      --suite-revision string            test suite revision. If not set SUITE_REVISION environment variable is used
      --test-type string                 test type. Allowed values: 'smoke', 'load' (default "smoke")
      --trufflehog-exclude-file string   path to the TruffleHog exclude-paths file. When set with --report-input trufflehog,
                                         exclusion pattern metrics are emitted alongside scan results.
                                         The file uses Go regexp syntax, one pattern per line (same format as --exclude-paths).
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

###### Auto generated by spf13/cobra on 4-Aug-2026
