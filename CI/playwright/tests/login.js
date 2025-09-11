import { test, expect } from '@playwright/test';

test('login with username and password', async ({ page, context }) => {
  const user = context.user || { user: 'admin', password: 'admin' };

  // Navigate to login page
  await page.goto('/login');
  
  // Fill in username and password
  await page.fill('input[name="user"]', user.user);
  await page.fill('input[name="password"]', user.password);
  
  // Click the login button
  await page.click('button[type="submit"]');

  // Wait for navigation to complete (handles React app redirects)
  await page.waitForLoadState('networkidle');

  // Check login result - could redirect to dashboard, setup guide, or stay on login for password update
  if (page.url().includes('/login')) {
    console.log('Redirected to password update page');
    await expect(page.locator('text=Update your password')).toBeVisible();
  } else if (page.url().includes('grafana-setupguide-app')) {
    // Hosted Grafana redirects to setup guide after login
    console.log('Redirected to setup guide (hosted Grafana)');
    await expect(page.locator('[aria-label="Profile"]')).toBeVisible();
  } else {
    // Standard Grafana dashboard
    await expect(page.locator('[aria-label="Profile"]')).toBeVisible();
    await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
  }
});

