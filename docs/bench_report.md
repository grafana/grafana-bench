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
      --bench-revision string              bench revision. If not provided BENCH_REVISION env var is used. 
                                           If not set, the current git revision is used (default "(devel)")
      --format string                      deprecated. Use --report-output instead
      --grafana-admin-password string      grafana admin user's password. Overridden by the GRAFANA_ADMIN_PASSWORD environment variable (default "admin")
      --grafana-admin-user string          grafana admin user name. Overridden by the GRAFANA_ADMIN_USER environment variable (default "admin")
      --grafana-url string                 grafana url. If not provided GRAFANA_URL env var is used
      --grafana-version string             grafana version. If not provided GRAFANA_VERSION env var is used.
                                           If not set, the version is retrieved from the grafana instance, if provided.
  -h, --help                               help for report
      --report-input string                report input format. Valid values are 'playwright' and 'go'
      --report-output string               format of the test execution report. Allowed values 'log', 'json' and 'text'.
                                            log' produced a structure log. 'json' produce a json object for each log line.
                                           'text' produced an human readable output (default "log")
      --run-id string                      test suite run id. If not specified, RUN_ID environment variable is used.
                                           If not set, an id is generated from the execution timestamp
      --run-metrics stringToString         test suite run custom metrics (default [])
      --run-metrics-prefix string          prefix to append to the suite run metric names
      --run-trigger string                 bench execution trigger (default "local")
      --suite-name string                  test suite name. If not specified, SUITE_NAME environment variable is used.
      --suite-run-metrics stringToString   deprecated use --run-metrics (default [])
      --suite-run-metrics-prefix string    deprecated. Use --run-metrics-prefix
      --test-suite-name string             deprecated. Use --suite-name instead
      --test-suite-run string              deprecated. Use --run-id
      --test-trigger string                deprecated. Use --run-trigger instead
      --test-type string                   test type. Allowed values: 'smoke', 'load' (default "smoke")
      --trigger string                     deprecated. Use --run-trigger instead
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

###### Auto generated by spf13/cobra on 7-Mar-2025
