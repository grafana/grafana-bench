
## Writing Playwright tests

Currently, there is no way to set the baseURL or executablePath of playwright via the command line. Instead, Bench will pass these values via Environment variable that will need to be referenced in the playwright.config.ts file of the project being tested.

process.env.GRAFANA_URL will be the same value passed as --grafana-url in the cli arguments.

process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH is set in the docker image of bench. It refers to the chromium executable on the image itself, currently /usr/bin/chromium. This is used because the playwright install command provided by playwright does not support alpine / musl.

```ts
// Include this into your playwright config
export default defineConfig({
  testDir: "./tests",
  use: {
    baseURL: process.env.GRAFANA_URL,
    launchOptions: {
      executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
    },
  },
});
```
