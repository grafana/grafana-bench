
## Writing Playwright tests

Currently, there is no way to set the baseURL or executablePath of playwright via the command line. Instead, Bench will pass these values via Environment variable that will need to be referenced in the playwright.config.ts file of the project being tested.

The following cli arguments will be passed through Bench and available in the Playwright config as environment variables via the `process.env`. They will be :

`--grafana-url` will be available as `process.env.GRAFANA_URL`
`--grafana-username` will be available as `process.env.GRAFANA_USERNAME`
`--grafana-password` will be available as `process.env.GRAFANA_PASSWORD`

`process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH` is set in the docker image of bench. It refers to the chromium executable on the image itself, currently "/usr/bin/chromium". This is used because the playwright install command provided by playwright does not support alpine / musl. If you do not include this in your config your tests may not run.

### Example Playwright config

Below is an example usage of the environment variables in a playwright config.

```ts
const pluginE2eAuth = `${dirname(require.resolve('@grafana/plugin-e2e'))}/auth`;

export default defineConfig({
  testDir: "./tests",
  use: {
    // will set the url of the grafana instance pass via cli params from bench
    baseURL: process.env.GRAFANA_URL,
    launchOptions: {
      // point playwright to preloaded chromium executable on bench docker image
      executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
    },
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
                 // username and password passed via cli params
                 // available as environment variables
                 user: process.env.GRAFANA_USERNAME,
                 password: process.env.GRAFANA_PASSWORD,
                 role: 'Admin',
            },
        },
        }
    ]
});
```

### Example Playwright command

```sh
bench test  \
  --grafana-url "http://host.docker.internal:3000" \
  --grafana-username "cool_user_name" \
  --grafana-password "test123" \
  --test-suite grafana-plugin-tests \
  --test-runner playwright \
  --pw-prepare-cmd "yarn install" \
  --pw-execute-cmd "yarn test" \
```
