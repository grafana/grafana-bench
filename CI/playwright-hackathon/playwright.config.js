import { dirname } from 'path';
import { defineConfig, devices } from '@playwright/test';

const pluginE2eAuth = `${dirname(require.resolve('@grafana/plugin-e2e'))}/auth`;

export default defineConfig({
  testDir: "./tests",
  use: {
    // will set the url of the grafana instance pass via cli params from bench
    baseURL: process.env.GRAFANA_URL || "http://localhost:3000",
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
          user: process.env.GRAFANA_ADMIN_USER || "admin",
          password: process.env.GRAFANA_ADMIN_PASSWORD || "admin",
          role: 'Admin',
        },
      },
    },

    // yarn playwright test --project=get-frontend-settings --ui
    {
      name: 'get-frontend-settings',
      testMatch: [/.*\.js/],
      use: {
        ...devices["Desktop Chrome"],
          user: {
            user: "admin",
            password: "admin",
          }
      },
    }

  ]
});
