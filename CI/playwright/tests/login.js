import { test, expect } from '@playwright/test';

test('login with username and password', async ({ page }) => {
  const username = process.env.GRAFANA_ADMIN_USER || 'admin';
  const password = process.env.GRAFANA_ADMIN_PASSWORD || 'admin';

  await page.goto('/login');

  await page.waitForSelector('[data-testid="data-testid Username input field"]', { state: 'visible' });
  await page.fill('[data-testid="data-testid Username input field"]', username);
  await page.fill('[data-testid="data-testid Password input field"]', password);
  await page.click('[data-testid="data-testid Login button"]');

  // Wait for navigation away from the login page
  await page.waitForURL(url => !url.pathname.startsWith('/login'), { timeout: 15000 }).catch(() => {});

  // Check for login errors
  const loginError = page.locator('[data-testid="alert-error"]');
  if (await loginError.isVisible()) {
    throw new Error('Login failed: ' + await loginError.textContent());
  }

  if (page.url().includes('/login')) {
    // Password update required is an acceptable post-login state
    await expect(page.locator('text=Update your password')).toBeVisible();
  } else {
    await expect(page.locator('[aria-label="Profile"], [data-testid="user-menu"]')).toBeVisible({ timeout: 10000 });
  }
});
