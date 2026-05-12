# Writing K6 API Tests

## Quickstart

1. **Start a Grafana instance to test against:**
   ```sh
   docker run -d --name=grafana -p 3000:3000 grafana/grafana
   ```

2. **Create a K6 test:**

   We're going to make a basic request to the Grafana instance and make sure it's running. K6 has support for TypeScript, so create a file called `check_grafana_instance.ts` with the following:

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

3. **Run the tests:**
   ```sh
   grafana-bench test \
     --service my-service \
     --service-version 1.0.0 \
     --service-url http://localhost:3000 \
     --test-runner k6 \
     --suite-name my-repo/k6 \
     --suite-path check_grafana_instance.ts
   ```

You should see output which looks like:

```text
CI/api_test.ts ... passed

Tests executed 1
Tests passed 1
Tests failed 0
Tests error 0

Tests suite passed
```

## Retrying failed tests

If a test can fail transiently — for example, because the service under test briefly returns 5xx during a deploy — you can ask bench to retry failed tests before giving up:

```sh
grafana-bench test \
  --test-runner k6 \
  --k6-retries 3 \
  --k6-retry-delay 10s \
  ... other flags ...
```

- `--k6-retries` — maximum number of retries after the initial run (default `0`, retries off).
- `--k6-retry-delay` — wait between attempts (default `0`). Accepts any Go duration (`500ms`, `5s`, `1m`).

Each retry re-runs the entire test file (including setup and teardown). JSON output for attempt 1 keeps the historical `/tmp/<name>.json` path; retry attempts write to `/tmp/<name>-attempt-N.json` so prior runs are preserved for postmortem.

### Reporting

Each test logs `attempts` and `maxAttempts` on its `msg=testRun` summary line. Status is reported as:

- `passed` — passed on the first attempt.
- `flaky` — failed at least once but passed within the retry budget. A `bench_test_run_flaky` metric is emitted.
- `failed` — failed every attempt.

The `totalDuration` for a retried test is the sum of wall-clock time across all attempts, so a test that keeps retrying shows an increasing duration.

> **Note:** `bench analyze` treats tests reported as `flaky` as not-a-defect by design. Enabling retries reduces the number of defects the analyzer can confirm — only tests that fail every attempt will surface as confirmed defects.

## Introduction

K6 tests are written in JavaScript and run in the Goja runtime. This is a small but important detail. While it looks like standard JavaScript, we don't have all of the capabilities of Node and K6 has added a few of its own.

For instance, we do not have access to writing or reading files, but we do have a super fast implementation for generating random data. Additionally K6 handles concurrency for us during load testing.

