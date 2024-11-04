# Principles of Bench
Bench is the glue that provides observability between local development, CI, and release pipelines. 

## A simple idea
At it's core, we wrap our testing tools and provide a standardized structured logging output for tests and test suites that looks like:

```sh
PLACEHOLDER
```

## Versioning
Since Bench runs in CI and as part of release pipelines we view it as critical software and thus follow a versioned release process. We DO NOT publish a `latest` package because we want users to be very aware of what version of Bench they are using when it is deployed and opt into upgrades manually.


## The Bench Value Proposition
1. Conventions for passing a Grafana instance to a test
2. Consistent failure modes for tests
3. Consistent structured output for results
4. Conventions for linking products, pipelines, test suites, and tests for downstream analysis and querying


## Values
* Stay out of the developers way
* Ensure portability


## Key Concepts
1. [test executors]
2. [passing variables]
3. [log output and formatting]

