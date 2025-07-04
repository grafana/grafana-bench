import { expect } from "@playwright/test";
import { test } from '@playwright/test';

test('fetch settings from /api/frontend/settings', async ({ request, baseURL }) => {
  // Make GET request to the settings endpoint
  const response = await request.get('/api/frontend/settings');

  // START HERE
  // 1. get auth working properly. double check the docs. see if the auth test is running first
  // 2. once request is working, get bench running in container command with latest image
  
  // Verify the response is successful
  expect(response.ok()).toBeTruthy();
  expect(response.status()).toBe(200);
  
  // Parse the JSON response
  const settingsData = await response.json();
  
  // Log the fetched settings object
  console.log('Settings data:', JSON.stringify(settingsData, null, 2));
  
  // Basic validation that we got an object back
  expect(settingsData).toBeDefined();
  expect(typeof settingsData).toBe('object');
  
  // Optional: Add specific assertions based on expected structure
  // Example assertions (uncomment and modify as needed):
  // expect(settingsData).toHaveProperty('theme');
  // expect(settingsData).toHaveProperty('apiVersion');
  // expect(settingsData.enabled).toBe(true);
  
  return settingsData;
});
