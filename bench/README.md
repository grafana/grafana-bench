# Bench library for testing


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
