import { dirname } from 'path';
import { defineConfig, devices } from '@playwright/test';

const pluginE2eAuth = `${dirname(require.resolve('@grafana/plugin-e2e'))}/auth`;

export default defineConfig({
  testDir: "./tests",
  use: {
    // will set the url of the grafana instance pass via cli params from bench
    baseURL: process.env.GRAFANA_URL || "http://localhost:3000",
    // Bot prevention countermeasures
    viewport: { width: 1920, height: 1080 },
    userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
    // Add realistic browser features
    javaScriptEnabled: true,
    acceptDownloads: true,
    // Slow down actions to appear more human
    actionTimeout: 10000,
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
      name: 'login',
      testMatch: [/.*\.js/],
      use: {
        ...devices["Desktop Chrome"],
        user: {
          user: process.env.GRAFANA_ADMIN_USER || "admin",
          password: process.env.GRAFANA_ADMIN_PASSWORD || "admin",
        }
      },
    }

  ]
});
