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
      --bench-revision string             grafana bench revision. If not set BENCH_REVISION environment variable is used.
                                          If not set, the current git revision is used (default (devel)  (default "(devel)")
      --dashboard string                  deprecated. Use run-dashboard
      --fetch-grafana-version string      Optional: Fetch Grafana version from API using provided credentials in 'user:password' format.
                                          Mutually exclusive with --service-version. Overridden by FETCH_GRAFANA_VERSION environment variable.
                                          Example: --fetch-grafana-version=admin:admin
      --format string                     deprecated. Use report-output
      --grafana-admin-password string     deprecated. Use --fetch-grafana-version for Grafana version fetching, or pass credentials directly to tests via environment variables (default "admin")
      --grafana-admin-user string         deprecated. Use --fetch-grafana-version for Grafana version fetching, or pass credentials directly to tests via environment variables (default "admin")
      --grafana-timeout duration          deprecated. Use --service-timeout (default 1m0s)
      --grafana-url string                deprecated. Use --service-url (default "http://localhost:3000")
      --grafana-version string            deprecated. Use --service-version
  -h, --help                              help for report
      --prometheus-metrics                send test suite run results to a prometheus remote write endpoint.
      --prometheus-password string        prometheus remote write password. If not set PROMETHEUS_PASSWORD environment variable is used.
      --prometheus-strict-lint            strict lint prometheus metrics. If set to true, will fail if metric does not pass linting
      --prometheus-timeout duration       prometheus remote write timeout. If not set PROMETHEUS_TIMEOUT environment variable is used.
      --prometheus-url string             prometheus remote write URL. If not set PROMETHEUS_URL environment variable is used.
      --prometheus-user string            prometheus remote write user. If not set PROMETHEUS_USER environment variable is used.
      --report-format string              deprecated. Use report-output (default "text")
      --report-input string               report input format. Valid values are 'playwright' and 'go'
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
      --run-trigger string                deprecated. Use run-stage. trigger of bench execution. For example, 'ci' or 'local'. (default "local")
      --service string                    REQUIRED. Name of the service being tested (e.g., 'grafana', 'loki', 'tempo', 'datasources'). Used for identifying which service the test results belong to in logs and metrics.
      --service-health-check              Perform a TCP health check on the service before running tests. Uses --service-url and --service-timeout.
      --service-timeout duration          timeout for waiting for the service to be live (default 1m0s)
      --service-url string                URL to the service being tested. Overridden by the SERVICE_URL environment variable (default http://localhost:3000) (default "http://localhost:3000")
      --service-version string            REQUIRED. Version of the service being tested (e.g., '11.0.0', '2.9.0'). Overridden by the SERVICE_VERSION environment variable.
      --suite-name string                 test suite name. If not specified, SUITE_NAME environment variable is used.
                                          Defaults to the last component of -suite-path.
                                          For example --suite--path path/to/testsuite will give a test suite name of 'testsuite'.
      --suite-revision string             test suite revision. If not set SUITE_REVISION environment variable is used
      --suite-run-metrics strings         deprecated use --run-metrics
      --suite-run-metrics-prefix string   deprecated. Use --run-metrics-prefix
      --test-report-format string         deprecated. Use report-output
      --test-suite-name string            deprecated. Use suite-name
      --test-suite-revision string        deprecated. Use suite-revision
      --test-trigger string               deprecated. Use run-stage (default "local")
      --test-type string                  test type. Allowed values: 'smoke', 'load' (default "smoke")
      --trigger string                    deprecated. Use run-stage (default "local")
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

###### Auto generated by spf13/cobra on 22-Jan-2026
