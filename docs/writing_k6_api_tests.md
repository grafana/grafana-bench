# Writing k6 API Tests

## Quickstart

### [Install Bench](index.md#installing-bench)

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


## Introduction

K6 tests are written in Javascript and run in the Doja runtime. This is a small
but important detail. While it looks like standard javascript, we don't have
all of the capabilities of Node and K6 has added a few of it's own.

For instance, we do not have access to writing or reading files, but we do have
a super fast implementation for generating random data. Additionally K6 handles
concurrency for us during load testing.

## Structure

An api tests suite is broken into 4 logical layers.

1. utilities for making requests
2. implementation of the api
3. utilities for tests
4. Implementation of tests

Directories:

``` shell
  /lib              # API implementation and utils
  /lib/config.ts
  /lib/dashboards.ts
  /lib/playlists.ts
  /lib/session.ts

  /tests            # test implementation and utils
  /tests/dashboards/dashboard_crud.ts
  /tests/playlists/playlist_crud.ts
```

This structure is still in development, however, we have a pretty good idea of
where we need boundaries for each of these pieces. We are actively working on
tooling to help us generate api implementations and ensuring we have these boundaries
will give us the ability to regenerate our api files as needed.

## Implementing an API

### API Helpers

### checks at the api layer

## Implementing a test

### setup

### authentication

### checks at the test layer

## A not on load testing
