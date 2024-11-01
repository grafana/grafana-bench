# Principles of Bench

## The Bench Value Proposition
1. Conventions for passing a Grafana instance to a test
2. Consistent failure modes for tests
3. Consistent structured output for results
4. Conventions for linking products, pipelines, test suites, and tests for downstream analysis and querying

Bench is the glue to deliver reliability and insights from our tests. Because of that we view Bench as critical software and thus follow a versioned release process. We DO NOT publish a `latest` package because we want users to be very aware of what version of Bench they are using when it is deployed.

## Values
* stay out of the developers way
* ensure portability


## Key Concepts
1. test executors
2. passing variables
3. log output and formatting