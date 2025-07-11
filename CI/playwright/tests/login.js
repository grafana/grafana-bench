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


  // if we're still on login page, check to see if we have the Update dialog
  if (page.url().includes('/login')) {
    console.log('Redirected to password update page');
    await expect(page.locator('text=Update your password')).toBeVisible();
  } else {
    await expect(page.locator('[aria-label="Profile"]')).toBeVisible();
    await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
  }
});

