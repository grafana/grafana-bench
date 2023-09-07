# Grafana Bench
Grafana bench is a tool to load test Grafana. It utilizes
https://github.com/grafana/grafana-build and https://github.com/grafana/grafana-api-tests to build and test a grafana on the platform and architecture of your choosing.

## Setup
### Dependencies
install the mage, k6, and docker for your OS

### Environment
Copy .env.sample to .env and set variables.
All of this can be overridden by command line.

## Usage
** Note, the first time you run one of these commands may take a while,
sequential runs will be faster due to caching.

`mage bench {test}`

where test is the path in the api tests repo inside the test/ folder. 
You can specify a folder or file. e.g. `mage bench dashboards` or ` mage bench dashboards/dashboard_create.js`

The bench command will handle building, provisioning, and testing the instance. The build step will ensure that the specified version exists in your local build cache so that it can be provisioned. If you specify a `GCP_CREDS_FILE` environment variable, this will utilize the remote buildcache and allow you to push/pull existing builds otherwise it will use your local buildcache.

You can change the version of grafana you're testing by setting the `GRAFANA_REVISION` environment variable either with `branch:{branchname}` or `commit:{commitSHA}`

When a run is performed, a state ID is generated and state is written to a folder. By default that is `{root}/work/provision/{stateID}` you can then use that state in successive runs.

If you're doing local development on a test suite or otherwise you can use the command `mage run` (When you ctrl + c the process will stop) and `STATE={stateID} mage test {test}` in another window. 

If you want to report your results to k6 cloud, set the `REPORT_CLOUD` environment variable to true and specify `K6_CLOUD_PROJECT` and `K6_CLOUD_TOKEN`

### Environment Variables / Flags
See https://github.com/grafana/grafana-bench/blob/main/bench/cfg.go


```.sh
# CLI options
GRAFANA_REVISION=branch:main
GRAFANA_ARCH=linux/amd64
PROVISION_DRIVER=local
PROVISION_STATE=

#GCP credentials
GCP_CREDS_FILE=

#K6
REPORT_CLOUD=false
K6_CLOUD_PROJECT=
K6_CLOUD_TOKEN=
```
