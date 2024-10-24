# Grafana Bench
Bench is a tool for executing e2e tests against an instance of Grafana.

## Value proposition
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
    `docker pull us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.2.3`

#### Github
1. Generate a set a github personal access token. Then run the command
    `gh auth token | docker login ghcr.io -u {YOUR_USERNAME} --password-stdin`
2. Pull the github container
    `docker pull ghcr.io/grafana/grafana-bench:v0.2.3`
    
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
### Write an API Test with K6
### Write a browser test with plugin-e2e framework and Playwright

## Diving deeper


1. See the help docs

    `grafana-bench help`

1. See the docs for `test` command

    `grafana-bench test --help`

1. Start a grafana instance to test against

    `docker run -d --name=grafana -p 3000:3000 grafana/grafana`

1. Create a K6 test
    We're going to make a basic request to the Grafana instance and make sure
    it's running. K6 has support for typescript, so create a file called `check_grafana_instance.ts`
    with the following:

    ```typescript

import { check } from 'k6';
import { http } from 'k6/http'

export const options = {
    scenarios: {
        api: {
          executor: 'shared-iterations',
        },
    },
};

export default function () {
    const res = http.request('GET', '<http://localhost:3000>');
    check(res, { 'status ok': res.status === 200 });
}
    ```

1. Run the tests

    ``` shell
    grafana-bench test --test-suite check_grafana_instance.ts
    ```

    You should see output which looks like

    ```shell

CI/api_test.ts ... passed

Tests executed 1
Tests passed 1
Tests failed 0
Tests error 0

Tests suite passed
    ```

## Next [Write some K6 API tests](writing_k6_api_tests.md)
