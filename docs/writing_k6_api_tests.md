# Writing k6 API Tests

## Introduction

K6 tests are written in Javascript and run in the Doja runtime. This is a small
but important detail because while what we are writing looks like javascript, we
don't have all of the capabilities of Node or the browser and have added a few
of our own.

For instance, we do not have access to writing or reading files, but we do have
a super fast implementation for generating random data.

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
