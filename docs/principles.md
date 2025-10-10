# Principles of Bench

Bench is the glue that provides testing observability between local development, CI, and release pipelines.

## A Simple Idea

At its core, we wrap our testing tools and provide a standardized structured logging output for tests and test suites. The spec is in progress.

**Documentation coming soon**

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

1. Conventions for passing Grafana credentials to a test as environment variables
2. Consistent structured output for results
3. Conventions for linking test results to products, pipelines, and test suites for analysis

## Project Values

* Aim for simplicity
* Stay out of the developer's way
* Ensure portability

## How Bench Works

Bench works by wrapping a testing tool, parsing its output, and producing structured logs that meet our [log format](#log-format).

## Test Executors

**Documentation coming soon**

## Passing Variables

**Documentation coming soon**

## Log Format

**Documentation coming soon**


