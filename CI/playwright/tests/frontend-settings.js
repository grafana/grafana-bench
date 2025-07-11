//test('fetch settings from /api/frontend/settings', async ({ request }) => {




//  // Make GET request to the settings endpoint
//  const response = await request.get('/api/frontend/settings');

//  // Verify the response is successful
//  expect(response.ok()).toBeTruthy();
//  expect(response.status()).toBe(200);
  
//  // Parse the JSON response
//  const settingsData = await response.json();
  
//  // Log the fetched settings object
//  console.log('Settings data:', JSON.stringify(settingsData, null, 2));
  
//  // Basic validation that we got an object back
//  expect(settingsData).toBeDefined();
//  expect(typeof settingsData).toBe('object');
  
//  return settingsData;
//});
