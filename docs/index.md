# Grafana Bench

Bench is a tool for executing e2e tests against an instance of Grafana.

## Quickstart tutorial

1. Prepare github credentials and sign into ghcr repository

    Generate a personal access token. Then run the command

    `gh auth token | docker login ghcr.io -u {YOUR_USERNAME} --password-stdin`

1. Install Bench
    For this tutorial, we want to get your running bench for local development quickly,
    however, in CI or for plugin-e2e tests we recommend using the Bench image as
    all dependencies for browser tests are provided for you.

    `go install github.com/grafana/grafana-bench`

    Bench is published to GAR and github container registry. It is recommended
    to use GAR as the GHCR image may be removed in the future.

    `docker pull us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.2.3`

    Images are tagged with the release version. To encourage versioning for use
    in ci pipelines, we don't produce a `latest` tag.

    Alternatively

    `docker pull ghcr.io/grafana/grafana-bench:v0.2.3`

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
