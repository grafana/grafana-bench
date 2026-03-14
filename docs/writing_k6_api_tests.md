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

## Introduction

K6 tests are written in JavaScript and run in the Goja runtime. This is a small but important detail. While it looks like standard JavaScript, we don't have all of the capabilities of Node and K6 has added a few of its own.

For instance, we do not have access to writing or reading files, but we do have a super fast implementation for generating random data. Additionally K6 handles concurrency for us during load testing.

