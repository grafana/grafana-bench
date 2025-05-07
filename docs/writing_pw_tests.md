# Writing Playwright tests

Grafana uses the [grafana/plugin-e2e](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/introduction) based on playwright to perform browser testing. Bench has a test executor for running playwright tests.

## Quickstart

1. [Install Bench](index.md#installing-bench)
2. [Passing the Grafana instance to the test](#passing-the-grafana-instance-to-the-test)
3. [Configuring a new project](#configuring-a-new-playwright-project)
4. [Adding to an existing project](#adding-to-an-existing-project)
5. [Run the test command](#run-the-tests)
6. [A minimal plugin-e2e smoke test](#a-minimal-plugin-e2e-smoke-test-example)
7. [Add your tests to CI](github_actions.md)

## Passing the Grafana instance to the test

Currently, there is no way to set the baseURL or executablePath of playwright via the command line. Instead, Bench will pass these values via Environment variable that will need to be referenced in the playwright.config.ts file of the project being tested.

The following cli arguments will be passed through Bench and available in the Playwright config as environment variables via the `process.env`. They will be :

`--grafana-url` will be available as `process.env.GRAFANA_URL`

`--grafana-admin-user` will be available as `process.env.GRAFANA_ADMIN_USER`

`--grafana-admin-password` will be available as `process.env.GRAFANA_ADMIN_PASSWORD`

Both of the examples below will show you how to configure your playwright tests to use these variables.

**NOTE plugin-e2e framework version 1.10.0 now looks for `process.env.GRAFANA_ADMIN_USER` and `process.env.GRAFANA_ADMIN_PASSWORD` by default, so you no longer need to configure those yourself.**

## Configuring a new playwright project

### Create a package.json

This is a minimul package.json to init your project. The important bits are the `setup` and `e2e` commands.
**note: plugin-e2e must be >= 1.10.0

```typescript
{
  "name": "YOUR_NAME",
  "version": "1.0.0",
  "repository": "git@github.com:grafana/YOUR_REPO",
  "author": "Author Name<author.email@domain.com>",
  "license": "MIT",
  "scripts": {
    "setup": "yarn install && playwright:install"
    "e2e": "playwright test",
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

### Create a playwright.config.ts

This is a minimal playwright config that will run the auth fixtures from the plugin-e2e package as a test and fail if authentication fails.

```typescript
import type { PluginOptions } from '@grafana/plugin-e2e';
import { defineConfig} from '@playwright/test';
import { dirname } from 'node:path';

const pluginE2eAuth = `${dirname(require.resolve('@grafana/plugin-e2e'))}/auth`;

export default defineConfig<PluginOptions>({
  testDir: './playwright/e2e',
  /* Run tests in files in parallel */
  fullyParallel: true,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  /* Opt out of parallel tests on CI. */
  workers: process.env.CI ? 2 : undefined,
  // Concise 'github' for CI, default 'list' when running locally
  reporter: process.env.CI ? 'html' : 'list',
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: process.env.GRAFANA_URL || `http://localhost:${process.env.PORT || 3000}`,

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'on',

  },
  projects: [
    {
      name: 'auth',
      // it is recommend to use the @grafana/plugin-e2e library to 
      // assist with auth
      testDir: pluginE2eAuth,
      testMatch: [/.*\.js/],
      use: {
        // This specifies the user for the test and is not related to Bench.
        user: {
          user: 'user',
          password: 'user',
          //role: 'Admin',
        },
      },
    }
  ]

}
```

## Adding to an existing project

### Add playwright

Plugin e2e **MUST** be >= 1.10.0

`yarn add grafana/plugin-e2e`

### Add setup and test scripts to your scripts block in package.json

```typescript
  "scripts": {
    "e2e": "playwright test",
  }
```

## A minimal plugin-e2e smoke test example

Derived from the [mssql datasource](https://github.com/grafana/grafana/pull/94977)

Check to verify that the plugin can be navigated to and we see the correct title.

```typescript
// mssql.spec.ts

import { test, expect } from '@grafana/plugin-e2e';

test('Smoke test: decoupled frontend plugin loads', async ({ createDataSourceConfigPage, page }) => {
  await createDataSourceConfigPage({ type: 'mssql' });

  await expect(await page.getByText('Type: Microsoft SQL Server', { exact: true })).toBeVisible();
  await expect(await page.getByRole('heading', { name: 'Connection', exact: true })).toBeVisible();
});
```

Register the test in the playwright.config.ts

```typescript
//playwright.config.ts
import { defineConfig, devices } from '@playwright/test';
import path, { dirname } from 'path';
import { PluginOptions } from '@grafana/plugin-e2e';
const testDirRoot = 'e2e/plugin-e2e/';
export default defineConfig<PluginOptions>({
  fullyParallel: true,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  /* Opt out of parallel tests on CI. */
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: process.env.GRAFANA_URL || `http://${process.env.HOST || 'localhost'}:${process.env.PORT || 3000}`,
    trace: 'retain-on-failure',
    provisioningRootDir: path.join(process.cwd(), process.env.PROV_DIR ?? 'conf/provisioning'),
  },
  projects: [
    // Login to Grafana with admin user and store the cookie on disk for use in other tests
    {
      name: 'authenticate',
      testDir: `${dirname(require.resolve('@grafana/plugin-e2e'))}/auth`,
      testMatch: [/.*\.js/],
    },
    // Register mssql test
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

### Install deps

`yarn setup`

### Run an instance of Grafana

This boots a docker container running grafana and mounts port 3000 to localhost so we can access it
`docker run --rm -p 3000:3000 grafana/grafana`

### Run the tests

Bench assumes the following defaults for specifying the grafana instance, so we don't need to add those to the command.

#### defaults

```sh
  --grafana-url "http://localhost:3000"
  --grafana-admin-user "admin"
  --grafana-admin-password "admin"
```

#### test command

```sh
docker run --rm \
  --network=host \
  --volume="./:/home/bench/tests/" \
  us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.4.1 test \
  --test-runner "playwright" \
  --test-suite-base "/home/bench/tests/" \
  --grafana-url "http://localhost:3000" \
  --pw-prepare-cmd "yarn install --frozen-lockfile; yarn playwright install" \
  --pw-execute-cmd "yarn e2e" \
  --test-env-vars "CI=true" \
    --verbose
```

#### Breakdown of the Bench command

1. `docker run --rm` invokes docker. `--rm` tells docker to remove the container when we're done
2. `--network=host` connects the docker container to the same network that the host is on. This is important as the the docker-compose file in the previous step mounts the grafana container to port 3000. So to make grafana accessible from the bench container, we need to connect the bench container to the same network.
3. `--volume="./:/home/bench/tests/"` mounts the current directory of the host machine inside the bench container. In this case, the checkout command from step 1 in the workflow grabs all of the plugin code and puts it in the current directory. So we're mounting everything inside the container in the `/home/bench/tests` directory
4. `us-docker.pkg.dev/grafanalabs-global/docker-grafana-bench-prod/grafana-bench:v0.4.1 test` says use the bench container tagged with bench:`v0.4.1`. The container specificies the bench binary as the default execution script, so `test` the subcommand and effectively runs `grafana-bench test`
5. `--test-runner "playwright"` tells the test command to use the playwright executor
6. `--test-suite-base "/home/bench/tests/"` specifies the directory we mounted the code in as the directory to execute the test runner from
7. `--pw-prepare-cmd "yarn install --frozen-lockfile; yarn playwright install"` specifies the two commands necessary to configure the e2e tests separated by a `;`. We do not currently support the `&&` operator, so you must use `;`. The first command installs yarn dependencies. The second installs playwright and dependencies
8. `--pw-execute-cmd "yarn e2e"` specifies the command to run the e2e tests.
9. `--test-env-vars "CI=true"` sets an environment variable to be passed to the test executor. Effectively `CI=true grafana-bench test ...`. It is common convention with playwright tests to use the `CI=true` flag
10. `--log-level DEBUG` sets the log level
