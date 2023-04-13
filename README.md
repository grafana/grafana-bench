# Grafana Bench
Grafana bench is a tool to load test Grafana. It utilizes
https://github.com/grafana/grafana-build and https://github.com/grafana/grafana-api-tests to build and test a grafana on the platform and architecture of your choosing.

## Dependencies
install the mage, k6, and docker for your OS

## Setup
`mage bootstrap`

## Usage
** Note, the first time you run one of these commands may take a while,
sequential runs will be faster due to build caching.

`mage test` - will build latest commit on main for your local system
architecture and run the k6 test suite against it

`COMMIT=k6-proof-of-concept mage test` - will grab latest commit from
k6-proof-of-concept branch. You can also use a full-length github commit hash
like `COMMIT=c116545e0ba005e10e318da96688bdae01439bf5 mage test`

Additionally you can specify custom configuration options by either putting a
custom.ini file in the project root or specifying a path to a custom.ini file.
`INI=/tmp/custom.ini mage test`

## Updates
The grafana-build and grafana-api-tests projects are developed separately from
this one. To update those projects run `mage update`

## Builds

When using this tool, we cache build artifacts in the artifacts/ directory.
If the executable is found, we skip building and run it

If you would like to build for a different architecture, you can do
that, however, running the build on your local machine is beyond the scope of
this project (for now) and probably won't lead to relevant test results. 

If you would like to create a build artifact for a different arch, you can provide the
ARCH flag

`ARCH=linux/amd64 mage build`

