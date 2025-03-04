import { dirname } from 'path';
import { defineConfig, devices, test as base } from '@playwright/test';

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
          user: process.env.GRAFANA_ADMIN_USER,
          password: process.env.GRAFANA_ADMIN_PASSWORD,
          role: 'Admin',
        },
      },
    },
    // npx playwright test --project=payload-size --ui
    {
      name: 'payload-size',

      testMatch: [/.*\.js/],
      use: {
        baseURL: "https://leeoniya.grafana.net",
        dashPath: "/d/bds35fot3cv7kb/empty?orgId=1&from=now-6h&to=now&timezone=browser",
        ...devices["Desktop Chrome"],
        // storageState: "playwright/.auth/admin.json",
        // launchOptions: {
        //   args: ['--js-flags=--expose-gc'],
        // },
      },
    }
  ]
});

export const test = base.extend({
  dashPath: ["", { option: true }],
  // panelLocator: [(fixtures) => fixtures.page.getByTestId("header-container"), { option: true }],
  // panel: [
  //   async ({ page }, use) => await use(page.getByTestId("header-container")),
  //   { option: true },
  // ],
  // panelByText: [undefined, { option: true }],
  // panelByTestId: ['header-container', { option: true }],
  /*
  // person: ['John', { option: true }],
  dashPath: '',
  panelLocator: [(page) => page.getByTestId("header-container"), { option: true }],
*/
});
