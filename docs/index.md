# Grafana Bench
Bench is a tool to provide test observability across the Grafana ecosystem. 

It works by:
1. Wrapping your e2e testing tool of choice with conventions for passing an instance of Grafana to the test
2. Massaging the test output into a standardized log format
3. Shipping the logs to a Loki instance
4. Dashboarding the results
4. Optionally performing slack alerts based on CodeOwners file

## **Bench is under active development.**

Our basic featureset and value proposition is defined along with a mostly stable api, however, we are trying to move fast to accomodate teams across Grafana and do release breaking changes. 

We recognize that as teams are using Bench in CI and for their release pipelines that Bench is critical infrastructure. In order to reduce the blast radius of any change, we provide semantically __versioned__ releases.

You can see our roadmap on [Github](https://github.com/orgs/grafana/projects/554)

## Table of Contents

### Quickstart
1. [Installing Bench](index.md#installing-bench)
1. [API tests with K6](writing_k6_api_tests.md)
1. [Broswer tests with plugin-e2e framework and Playwright](writing_pw_tests.md)

### Additional usecases
1. [Configuring CI](github_actions.md)
1. [Implementing a pipeline in Jsonnet](libsonnet.md)
1. [Configuring slack notifications with Codeowners](notifications.md)

#### Deep Dive

1. [Principles of Bench](principles.md)
1. [Log Format](princples.md#log_format)
1. [Product DNA](https://docs.google.com/document/d/1rs1RN8UHKAowcQX-cqJ5ilhOnykAMUSoqakWICOGtcY/edit?tab=t.0#heading=h.soj854l81770)

## Installing Bench
Bench can be run natively or in a docker container. We recommend docker for portability, however, using the binary may be beneficial when developing tests locally.

### Docker
We currently publish the docker image to both GAR and github packages. Some users have reported issues pulling the container from github. So we currently recommend using GAR. If you run into issues with either of these, please reach out to us in #grafana-bench.

#### Google Artifact Registry (GAR)
Fetch the container from Google Artifact registry.

    docker pull us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.2.4`

#### Github

1. Generate a set a github personal access token. Then run the command
    `gh auth token | docker login ghcr.io -u {YOUR_USERNAME} --password-stdin`

2. Pull the github container
    `docker pull ghcr.io/grafana/grafana-bench:v0.2.4`

3. Verify installation
`docker run --rm us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.2.4 --help`
    
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

3. Set GOPRIVATE to ensure we go straight to github instead of go module proxy.

    We recommend adding this to your ~/.profile or wherever your terminal config lives.

    `export GOPRIVATE=github.com/grafana/grafana-bench`


4. Install Bench

    Get the latest release version from github.com/grafana/grafana-bench

    `go install github.com/grafana/grafana-bench@v<VERSION>`. 

5. Verify installation

   `grafana-bench --help`

### Next steps

Write an [API test with K6](writing_k6_api_tests.md) or a [Browser test with Playwright and plugin-e2e](writing_pw_tests.md)