# Writing Playwright tests

Grafana uses the grafana/plugin-e2e which is based on playwright to browser test Grafana. We wrote a playwright executor for Bench to run playwright tests.

## Quickstart

### Grab the docker image
This is the Bench image to run the tests.
`docker pull ghcr.io/grafana/grafana-bench:v0.2.4`

### Create a package.json
This is a minimul package.json to init your project. The important bits are the `setup` and `e2e` commands.
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
    "@grafana/plugin-e2e": "^1.8.3",
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
    // This is used to authenticate the user before running the tests.
    // useful since the e2e CI tests run a different admin user
    // see https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/use-authentication
    grafanaAPICredentials: {
      user: process.env.GRAFANA_USER || "admin",
      password: process.env.GRAFANA_PASSWORD || "admin",
    },

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

### Install deps
`yarn setup`

### Run an instance of Grafana
This boots a docker container running grafana and mounts port 3000 to localhost so we can access it
`docker run --rm -p 3000:3000 grafana/grafana`

### Run the tests
This command does the following:
* sets the network to use your system network allowing the container to reach Grafana at localhost:3000
* mounts the current directory (we're assuming you're running this from project root) inside the container in /home/bench/tests
* runs the bench test command
* sets the test directory to the location we mounted
* defines `yarn setup` as the playwright command to setup the tests
* defines `yarn e2e` as the playwright command to run the tests
* sets the grafana url to localhost:3000 (the Grafana container we started)
* sets grafana username to "admin"
* sets grafana password to "admin"

```sh
docker run \
  --network=host \
  --volume="./:/home/bench/tests/" \
  ghcr.io/grafana/grafana-bench:v0.2.4 test \
  --test-suite-base "/home/bench/tests/" \
  --test-runner "playwright" \
  --pw-prepare-cmd "yarn setup" \
  --pw-execute-cmd "yarn e2e" \
  --grafana-url "http://localhost:3000" \
  --grafana-username "admin" \
  --grafana-password "admin" \
  --verbose
```

## Adding Bench + Playwright support to an existing project

### Add playwright
`yarn add grafana/plugin-e2e`

### Add setup and test scripts to your scripts block in package.json
```typescript
  "scripts": {
    "setup": "yarn install && playwright:install"
    "e2e": "playwright test",
  }
```




Currently, there is no way to set the baseURL or executablePath of playwright via the command line. Instead, Bench will pass these values via Environment variable that will need to be referenced in the playwright.config.ts file of the project being tested.

The following cli arguments will be passed through Bench and available in the Playwright config as environment variables via the `process.env`. They will be :

`--grafana-url` will be available as `process.env.GRAFANA_URL`
`--grafana-username` will be available as `process.env.GRAFANA_USERNAME`
`--grafana-password` will be available as `process.env.GRAFANA_PASSWORD`

`process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH` is set in the docker image of bench. It refers to the chromium executable on the image itself, currently "/usr/bin/chromium". This is used because the playwright install command provided by playwright does not support alpine / musl. If you do not include this in your config your tests may not run.
