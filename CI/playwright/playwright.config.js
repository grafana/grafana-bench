import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: "./tests",
  use: {
    // will set the url of the grafana instance passed via cli params from bench
    baseURL: process.env.GRAFANA_URL || "http://localhost:3000",
    viewport: { width: 1920, height: 1080 },
    javaScriptEnabled: true,
    acceptDownloads: true,
    actionTimeout: 10000,
  },
  projects: [
    {
      name: 'login',
      testMatch: [/.*\.js/],
      use: {
        ...devices["Desktop Chrome"],
      },
    },
  ],
});
