# Principles of Bench
Bench is the glue that provides observability between local development, CI, and release pipelines. 

## A simple idea
At it's core, we wrap our testing tools and provide a standardized structured logging output for tests and test suites. The spec is in progress. 

**Documentation coming soon

## Versioning
Since Bench runs in CI and as part of release pipelines we view it as critical software and thus follow a versioned release process. We DO NOT publish a `latest` package because we want users to be very aware of what version of Bench they are using when it is deployed and opt into upgrades manually.

## Table of Contents
1. [Versioning](#versioning)
2. [Value Proposition](#the-bench-value-proposition)
3. [Project Values](#project-values)
4. [How Bench Works](#how-bench-works)
1. [Test Executors](#test-executors)
2. [Passing Variables](#passing-variables)
3. [Log output and formatting](#log-output)

## The Bench Value Proposition
1. Conventions for passing a Grafana credentials to a test as environment variables
2. Consistent failure modes for tests
3. Consistent structured output for results
4. Conventions for linking products, pipelines, test suites, and tests for downstream analysis and querying

## Project Values
* Aim for simplicity
* Stay out of the developers way
* Ensure portability

## How Bench Works 
Bench works by wrapping a testing tool, parsing it's output, and producing structured logs that meet our [log format](#log-format)

## Test Executors
Bench 

## Passing Variables

## Log Format


