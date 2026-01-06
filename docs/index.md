# Grafana Bench

Bench is a tool to provide test observability across the Grafana ecosystem.


Latest Version: v0.6.11

## Background

Grafana can be built for many architectures, operating systems, and configured with many options
and plugins. We're currently doing most of our testing in CI, which only tests a single architecture, but
defects can occur at the packaging and configuration stages as well. The Bench project aims to improve our
test coverage by providing all of the tools and glue necessary to write and observe e2e tests for Grafana.

For a deep dive see: [Delivery Targets and the need for Testing Observability](https://docs.google.com/document/d/1pQoU3sccwayVK_LLzQUnOQLq-0jduH-4FibpdaAztX8/edit?tab=t.0)

### What we're trying to accomplish

* Increase the test coverage of Grafana by building reusable tests
* Empower teams to author, register, and receive timely feedback from their tests across delivery pipelines with as little friction as possible.
* Increase developer trust in integration and e2e tests across Grafana
* Observe the state of test across all components of Grafana

### How it works

The Bench CLI can be used either as a test executor OR as a reporter. The basics are:

In reporter mode:

1. run your test and write the output to a file
2. pass the file to the bench cli reporter command
3. standardized test output is sent to a loki instance
4. results are available in loki-ops
5. Optionally, slack alerts can be configured based on a CODEOWNERS file.

In executor mode:

1. run your test with the becnch cli exector command. output is parsed and formated automatically
2. output is sent to loki instance
3. results are available in loki-ops
4. Optionally, slack alerts can be configured based on a CODEOWNERS file.

## **Bench is under active development.**

Our basic feature set and value proposition is defined along with a mostly stable API, however, we are trying to move fast to accommodate teams across Grafana and do release-breaking changes.

We recognize that as teams are using Bench in CI and for their release pipelines that Bench is critical infrastructure. In order to reduce the blast radius of any change, we provide semantically **versioned** releases.

You can see our roadmap on [Github](https://github.com/orgs/grafana/projects/554)

## Table of Contents

### Quickstart

1. [Installing Bench](index.md#installing-bench)
2. [API tests with K6](writing_k6_api_tests.md)
3. [Broswer tests with plugin-e2e framework and Playwright](writing_pw_tests.md)

### Use cases

One of the key goals of Bench is to provide portability. We do this by making bench part of the development workflow- running tests locally during development, in CI as part of the SDLC, and finally in release pipelines as part of release.

1. [Configuring CI](github_actions.md)
2. [Implementing a pipeline in Jsonnet](libsonnet.md)
3. [Configuring slack notifications with Codeowners](notifications.md)
4. [Configuring a hosted Grafana instance for e2e tests](configuring_an_instance.md)
5. [Performing load testing on Grafana](load_testing.md)

#### Deep Dive

1. [Principles of Bench](principles.md)
2. [Log Format](princples.md#log_format)
3. [Product DNA](https://docs.google.com/document/d/1rs1RN8UHKAowcQX-cqJ5ilhOnykAMUSoqakWICOGtcY/edit?tab=t.0#heading=h.soj854l81770)

## Installing Bench

Bench can be run as a native command or in a docker container. We recommend docker for portability, however, using the binary may be beneficial when developing tests locally.

### Docker

We currently publish the docker image to both GAR and github packages. Some users have reported issues pulling the container from github. So we currently recommend using GAR. If you run into issues with either of these, please reach out to us in #grafana-bench.

#### Google Artifact Registry (GAR)

Fetch the container from Google Artifact registry.

    docker pull us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.6.11

#### Github

1. Generate a set a github personal access token. Then run the command
    `gh auth token | docker login ghcr.io -u {YOUR_USERNAME} --password-stdin`

2. Pull the github container
    `docker pull ghcr.io/grafana/grafana-bench:v0.6.11

3. Verify installation
`docker run --rm us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.6.11 --help`

### Binary / Go Package

We do not currently ship a binary, but you can install directly from the go project.

1. Make sure you have a [Go environment configured](https://go.dev/doc/install)
2. Configure [github to authenticate with SSH keys](https://docs.github.com/en/authentication/connecting-to-github-with-ssh).
3. Configure github to use ssh instead of https

   Add this block to ~/.gitconfig or wherever your configuration is located.

```ini
[url "git@github.com:"]
 insteadOf = https://github.com/
```

4. Set GOPRIVATE to ensure we go straight to github instead of go module proxy.

    We recommend adding this to your ~/.profile or wherever your terminal config lives.

    `export GOPRIVATE=github.com/grafana/grafana-bench`

5. Install Bench

    Get the latest release version from github.com/grafana/grafana-bench

    `go install github.com/grafana/grafana-bench@v<VERSION>`.

6. Verify installation

   `grafana-bench --help`

### Next steps

Write an [API test with K6](writing_k6_api_tests.md) or a [Browser test with Playwright and plugin-e2e](writing_pw_tests.md)
