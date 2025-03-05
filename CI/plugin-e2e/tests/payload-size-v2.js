import { expect } from "@playwright/test";
import { test } from "../playwright.config";
import * as fs from 'fs';

test("payload-size", { tag: '@performance' }, async ({ page, dashPath }) => {
  let inflatedSize = 0;
  let transferSize = 0;
  let requests = 0;

  page.on('console', msg => console.log(msg.text()));

  // Use a semaphore pattern to track active response processing
  let activeResponseCount = 0;
  let responsesComplete = false;
  let resolveAllResponses;

  // Create a promise we can resolve when all responses are done
  const allResponsesProcessed = new Promise(resolve => {
    resolveAllResponses = resolve;
  });

  const addSize = async (response) => {
    try {
      if (response.status() === 200) {
        activeResponseCount++;

        try {
          // Process the response
          const body = await response.body();
          inflatedSize += body.length;

          const sizes = await response.request().sizes();
          transferSize += sizes.responseBodySize + sizes.responseHeadersSize;
          requests++;
        } catch (err) {
          console.log('Error processing response:', err.message);
        } finally {
          // Decrement the counter and check if all responses are processed
          activeResponseCount--;
          if (responsesComplete && activeResponseCount === 0) {
            resolveAllResponses();
          }
        }
      }
    } catch (error) {
      console.log('Error in addSize handler:', error.message);
    }
  };

  page.on("response", addSize);

  let start = performance.now();
  await page.goto(dashPath);
  let el = page.getByTestId("header-container");
  await el.waitFor();
  await expect(el).toBeVisible();

  // Wait for network activity to settle
  await page.waitForLoadState('networkidle');

  // Signal that no more responses will be coming
  responsesComplete = true;

  // If there are no active responses, resolve immediately
  if (activeResponseCount === 0) {
    resolveAllResponses();
  }

  // Wait for all active responses to complete processing
  await allResponsesProcessed;

  let end = performance.now();

  // Remove the listener after all responses are processed
  page.removeListener("response", addSize);

  let client = await page.context().newCDPSession(page);
  await client.send('HeapProfiler.collectGarbage');
  let usedJSHeapSize = (await client.send("Runtime.getHeapUsage")).usedSize;

  // Create performance data object
  const performanceData = {
    boot: Math.round(end - start),
    inflatedSizeMB: +(inflatedSize / 1000 / 1000).toFixed(1),
    transferSizeMB: +(transferSize / 1000 / 1000).toFixed(1),
    requests: requests,
    usedJSHeapSize: +(usedJSHeapSize / 1000 / 1000).toFixed(1),
  };

  filename = "/tmp/asset-metrics.json"
  
  // Write the data to file
  fs.writeFileSync(filename, JSON.stringify(performanceData, null, 2));
  
  // Still log to console for debugging
  console.log(`Performance data written to ${filename}`);
  console.log(performanceData);
  
  client.detach();
});
