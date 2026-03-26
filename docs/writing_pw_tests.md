# Writing Playwright Tests

Bench has a test executor for running Playwright tests. It works well with the [grafana/plugin-e2e](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/introduction) framework for browser-based testing against a Grafana instance.

## Passing the Grafana instance to the test

There is no way to set the `baseURL` or `executablePath` of Playwright via the command line. Instead, bench passes these values as environment variables that you reference in `playwright.config.ts`:

- `--service-url` is available as `process.env.GRAFANA_URL`
- Any `--test-env KEY=VALUE` pairs are available as `process.env.KEY`

For authentication, pass credentials via `--test-env`:

```sh
--test-env GRAFANA_ADMIN_USER=admin --test-env GRAFANA_ADMIN_PASSWORD=admin
```

> **Note:** plugin-e2e >= 1.10.0 reads `process.env.GRAFANA_ADMIN_USER` and `process.env.GRAFANA_ADMIN_PASSWORD` by default, so you no longer need to wire those up yourself.

## Configuring a new Playwright project

### package.json

```json
{
  "name": "YOUR_NAME",
  "version": "1.0.0",
  "scripts": {
    "setup": "yarn install && playwright install chromium",
    "e2e": "playwright test"
  },
  "dependencies": {
    "@playwright/test": "^1.42.1",
    "playwright": "^1.42.1"
  },
  "devDependencies": {
    "@grafana/plugin-e2e": "^1.10.0",
    "@types/node": "^22.7.7"
  }
}
```

> plugin-e2e must be >= 1.10.0

### playwright.config.ts

A minimal config that runs plugin-e2e auth fixtures and fails if authentication fails:

```typescript
import type { PluginOptions } from '@grafana/plugin-e2e';
import { defineConfig } from '@playwright/test';
import { dirname } from 'node:path';

const pluginE2eAuth = `${dirname(require.resolve('@grafana/plugin-e2e'))}/auth`;

export default defineConfig<PluginOptions>({
  testDir: './playwright/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 2 : undefined,
  reporter: process.env.CI ? 'html' : 'list',
  use: {
    baseURL: process.env.GRAFANA_URL || `http://localhost:${process.env.PORT || 3000}`,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'on',
  },
  projects: [
    {
      name: 'auth',
      testDir: pluginE2eAuth,
      testMatch: [/.*\.js/],
      use: {
        storageState: `playwright/.auth/${process.env.GRAFANA_ADMIN_USER ?? 'admin'}.json`,
        user: {
          user: 'user',
          password: 'user',
        },
      },
    },
  ],
});
```

## Adding to an existing project

Install plugin-e2e:

```sh
yarn add @grafana/plugin-e2e
```

Add scripts to your `package.json`:

```json
"scripts": {
  "e2e": "playwright test"
}
```

## A minimal plugin-e2e smoke test example

Derived from the [mssql datasource](https://github.com/grafana/grafana/pull/94977) — checks that the plugin can be navigated to and shows the correct title:

```typescript
// mssql.spec.ts
import { test, expect } from '@grafana/plugin-e2e';

test('Smoke test: decoupled frontend plugin loads', { tag: '@grafana-bench' }, async ({ createDataSourceConfigPage, page }) => {
  await createDataSourceConfigPage({ type: 'mssql' });

  await expect(await page.getByText('Type: Microsoft SQL Server', { exact: true })).toBeVisible();
  await expect(await page.getByRole('heading', { name: 'Connection', exact: true })).toBeVisible();
});
```

The `@grafana-bench` tag lets you run only bench tests: `playwright test --grep @grafana-bench`

Register the test in `playwright.config.ts`:

```typescript
import { defineConfig, devices } from '@playwright/test';
import path, { dirname } from 'path';
import { PluginOptions } from '@grafana/plugin-e2e';

const testDirRoot = 'e2e/plugin-e2e/';

export default defineConfig<PluginOptions>({
  fullyParallel: true,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: process.env.GRAFANA_URL || `http://${process.env.HOST || 'localhost'}:${process.env.PORT || 3000}`,
    trace: 'retain-on-failure',
    provisioningRootDir: path.join(process.cwd(), process.env.PROV_DIR ?? 'conf/provisioning'),
  },
  projects: [
    {
      name: 'authenticate',
      testDir: `${dirname(require.resolve('@grafana/plugin-e2e'))}/auth`,
      testMatch: [/.*\.js/],
    },
    {
      name: 'mssql',
      testDir: path.join(testDirRoot, '/mssql'),
      use: {
        ...devices['Desktop Chrome'],
        storageState: 'playwright/.auth/admin.json',
      },
      dependencies: ['authenticate'],
    },
  ],
});
```

## Running the tests

### 1. Install dependencies

```sh
yarn setup
```

### 2. Start Grafana

```sh
docker run --rm -p 3000:3000 grafana/grafana
```

### 3. Run bench

```sh
docker run --rm \
  --network=host \
  --volume="./:/tests/" \
  ghcr.io/grafana/grafana-bench-playwright:v1.0.5 test \
  --service grafana \
  --service-url "http://localhost:3000" \
  --service-version latest \
  --test-runner playwright \
  --test-type smoke \
  --suite-path /tests \
  --suite-name my-repo/e2e \
  --run-stage local \
  --report-output log \
  --pw-prepare "yarn install --frozen-lockfile; yarn playwright install chromium" \
  --pw-execute "yarn e2e" \
  --test-env "CI=true" \
  --log-level debug
```

**Command breakdown:**

1. `docker run --rm` — removes the container when done
2. `--network=host` — connects the bench container to the host network so it can reach Grafana on port 3000
3. `--volume="./:/tests/"` — mounts the current directory into the container at `/tests`
4. `ghcr.io/grafana/grafana-bench-playwright:v1.0.5 test` — uses the Playwright variant of the bench image (includes browsers and system dependencies)
5. `--service-url "http://localhost:3000"` — passed to your tests as `process.env.GRAFANA_URL`
6. `--pw-prepare` — commands to run before the test (use `;` to separate, `&&` is not supported)
7. `--pw-execute` — command to run the tests
8. `--test-env "CI=true"` — sets an environment variable passed to the test executor

> **Note:** Use `grafana-bench-playwright` (not `grafana-bench`) for Playwright tests. It includes the browser binaries and system dependencies.

---

## Troubleshooting

### "su: Authentication failure" / "Failed to install browsers"

**Problem:**
```
Installing dependencies...
Switching to root user to install dependencies...
Password: su: Authentication failure
Failed to install browsers
```

**Cause:** The `--with-deps` flag tries to install system packages via `apt-get`, which requires root. The container runs as a non-root user.

**Solution:** Remove `--with-deps`:

```bash
# ❌ Requires root
--pw-prepare "yarn install; yarn playwright install --with-deps chromium"

# ✅ Downloads browsers only
--pw-prepare "yarn install; yarn playwright install chromium"
```

The `grafana-bench-playwright` image already includes all system dependencies. You only need to download browser binaries, which doesn't require root.

### Playwright version mismatches

**Problem:** Browsers are re-downloaded every run, or version mismatch errors appear.

**How it works:** The Docker image bundles browsers for a specific Playwright version. When your `package.json` specifies a different version, running `playwright install` downloads the matching browsers to `~/.cache/ms-playwright`.

**Solution:** This is expected. Just run `playwright install chromium` (without `--with-deps`) in your prepare command and it will handle version alignment automatically.

### "Executable doesn't exist"

Make sure your `--pw-prepare` command includes `playwright install`. The browser binaries must be downloaded even though system dependencies are pre-installed.

### Permission errors writing to cache

Ensure you're using the `grafana-bench-playwright` image. If you encounter volume mount permission issues between your host user and the container's `pwuser`, try removing and re-creating the volume mount.
