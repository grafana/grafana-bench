import { dirname } from 'path';
import { defineConfig, devices } from '@playwright/test';
const pluginE2eAuth = `${dirname(require.resolve('@grafana/plugin-e2e'))}/auth`;

export default defineConfig({
    testDir: process.env.PLAYWRIGHT_TEST_DIR,
    retries: 0,
    fullyParallel: true,
    headless: true,
    timeout: 5000,
    reporter: [
        ['json', { outputFile: './playwright-report.json' }],
    ],
    use: {
        baseURL: process.env.PLAYWRIGHT_BASE_URL,
        launchOptions: {
            executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH,
        }
    },
    projects: [
        {
            name: 'auth',
            testDir: pluginE2eAuth,
            testMatch: [/.*\.js/],
        },
        {
            name: 'run-tests',
            use: {
                ...devices['Desktop Chrome'],
                storageState: 'playwright/.auth/admin.json',
            },
            dependencies: ['auth'],
        },
    ],
});