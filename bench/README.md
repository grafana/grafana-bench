# Grafana Bench
Grafana bench is a tool to load test Grafana. It utilizes
https://github.com/grafana/grafana-build and https://github.com/grafana/grafana-api-tests to build and test a grafana on the platform and architecture of your choosing.

## how it works
* Bench is a wrapper around building and testing grafana using github.com/grafana/grafana-build and github.com/grafana/grafana-api-tests. 

* We use mage as a shortcut for building our own CLI tool and utilize
    environment variables for telling it what we want to do

* This project is active in helping to developer grafana-api-tests, however,
    that repo should be treated like a separate project. In other words, any
    changes to that repo should make sense on their own without bench

## guidelines
* top level commands should get their own file and should return an error. Those
tasks should be wrapped in ../Magefile.go when adding new tasks

* each sub command should make a call to ResolveConfig. That method is
    idempotent and will not run if the config is already resolved


## Setup
### Dependencies
A. Install mage, k6, and docker for your OS
B. Make sure you have access to github.com/grafana/grafana-api-tests. If you need to do any javascript bundling for your tests, you may need node + yarn

### Config
Copy .env.sample to .env and set variables.
These can be overridden by environment variables.

## Usage
** Note, the first time you run one of these commands may take a while,
sequential runs will be faster due to caching of builds etc.

### Testing an already running Grafana
By default we assume Grafana is running on localhost:3000 with username admin/admin.

### Using the build pipeline

`mage bench {test}`

where test is the path in the api tests repo inside the test/ folder. 
You can specify a folder or file. e.g. `mage bench dashboards` or ` mage bench dashboards/dashboard_create.js`

The bench command will handle building, provisioning, and testing the instance. The build step will ensure that the specified version exists in your local build cache so that it can be provisioned. If you specify a `GCP_CREDS_FILE` environment variable, this will utilize the remote buildcache and allow you to push/pull existing builds otherwise it will use your local buildcache.

You can change the version of grafana you're testing by setting the `GRAFANA_REVISION` environment variable either with `branch:{branchname}` or `commit:{commitSHA}`

When a run is performed, a state ID is generated and state is written to a folder. By default that is `{root}/work/provision/{stateID}` you can then use that state in successive runs.

If you're doing local development on a test suite or otherwise you can use the command `mage run` (When you ctrl + c the process will stop) and `STATE={stateID} mage test {test}` in another window. 

If you want to report your results to k6 cloud, set the `REPORT_CLOUD` environment variable to true and specify `K6_CLOUD_PROJECT_ID` and `K6_CLOUD_TOKEN`

### Environment Variables / Flags
See https://github.com/grafana/grafana-bench/blob/main/bench/cfg.go


```.sh
# CLI options
export GRAFANA_REVISION=branch:main
export GRAFANA_ARCH=linux/amd64
export PROVISION_DRIVER=local
export PROVISION_STATE=

# GCP credentials
export GCP_CREDS_FILE=

# K6
export REPORT_CLOUD=false
export K6_CLOUD_PROJECT_ID=
export K6_CLOUD_TOKEN=

# Grafana
export GRAFANA_ADDRESS=http://localhost:3000
export GRAFANA_USER=admin
export GRAFANA_PASSWORD=admin
```
