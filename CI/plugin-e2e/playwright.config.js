import { dirname } from 'path';
import { defineConfig } from '@playwright/test';

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
