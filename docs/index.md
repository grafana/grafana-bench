# Grafana Bench
Bench is a tool to provide test observability across the Grafana ecosystem. It works by:
1. Wrapping your e2e testing tool of choice with conventions for testing Grafana
2. Massaging the test output into a standardized log format
3. Shipping the logs to a Loki instance
4. Dashboarding the results to provide insights
4. 

[Product DNA](https://docs.google.com/document/d/1rs1RN8UHKAowcQX-cqJ5ilhOnykAMUSoqakWICOGtcY/edit?tab=t.0#heading=h.soj854l81770)

## Quickstart
1. [Installing Bench](index.md#installing-bench)
2. [API tests with K6](writing_k6_api_tests.md)
3. [Broswer tests with plugin-e2e framework and Playwright](writing_pw_tests.md)

## The Bench Value Proposition
1. Conventions for passing a Grafana instance to a test
2. Consistent failure modes for tests
3. Consistent structured output for results
4. Conventions for linking products, pipelines, test suites, and tests for downstream analysis and querying

Bench is the glue to deliver reliability and insights from our tests. Because of that we view Bench as critical software and thus follow a versioned release process. We DO NOT publish a `latest` package because we want users to be very aware of what version of Bench they are using when it is deployed.

## Installing Bench
Bench can be run as a Go binary natively or in a docker container. We recommend a docker for portability, however, using the binary may be beneficial when developing tests locally.

### Docker
We currently publish the docker image to both GAR and github packages. Some users have reported issues pulling the container from github. So we currently recommend using GAR. If you run into issues with either of these, please reach out to use in #grafana-bench.

#### GAR

`docker pull us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.2.4`

#### Github

1. Generate a set a github personal access token. Then run the command
    `gh auth token | docker login ghcr.io -u {YOUR_USERNAME} --password-stdin`

2. Pull the github container
    `docker pull ghcr.io/grafana/grafana-bench:v0.2.4`

3. Verify installation
`docker run --rm us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.2.4 --help`
    
### Go Package
1. Make sure you have a Go environment configured and install Go
2. Configure github to authenticate with SSH keys. Add your local key to your github profile and ensure you have this line in your ~/.gitconfig
```ini
[url "git@github.com:"]
	insteadOf = https://github.com/
```
3. Set GOPRIVATE to ensure we go straight to github instead of go module proxy.
Run `export GOPRIVATE=github.com/grafana/grafana-bench`

We recommend adding this to your ~/.profile or wherever your terminal config lives.

4. Install Bench
Get the latest release version from github.com/grafana/grafana-bench

`go install github.com/grafana/grafana-bench@v<VERSION>`. 

5. Verify installation
`grafana-bench -h`

## Next Steps

### Core concepts
1. Bench is a wrapper around test executors
### [API tests with K6](writing_k6_api_tests.md)
### [Broswer tests with plugin-e2e framework and Playwright](writing_pw_tests.md)