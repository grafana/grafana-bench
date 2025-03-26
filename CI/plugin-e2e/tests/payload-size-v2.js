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
  await page.waitForLoadState('networkidle', {timeout: 50_000});

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
  const metricsWithLabels = {
    boot: {
      value: Math.round(end - start),
    },
    inflatedSizeMB: {
      value: +(inflatedSize / 1000 / 1000).toFixed(1),
    },
    transferSizeMB: {
      value: +(transferSize / 1000 / 1000).toFixed(1),
    },
    requests: {
      value: requests,
    },
    usedJSHeapSize: {
      value: +(usedJSHeapSize / 1000 / 1000).toFixed(1),
    }
  };

  console.log(performanceData);

  // Write json data to file
  const textExpositionData = convertToPrometheusFormat(metricsWithLabels);
  console.log(textExpositionData);
  fs.writeFileSync('/tmp/asset-metrics.txt', textExpositionData);

  client.detach();
}, { timeout: 60000 });


// DISCLAIMER. I had claude write all of this so it's probably terrible.

/**
 * Converts performance data with integrated labels to Prometheus exposition text format
 * https://github.com/prometheus/docs/blob/main/content/docs/instrumenting/exposition_formats.md#text-format-example
 * @param {Object} metrics - Object containing metrics with their values and labels
 * @returns {string} - Formatted Prometheus exposition text
 */
function convertToPrometheusFormat(metrics) {
  const lines = [];
  const timestamp = Date.now();
  
  // Process each metric
  for (const [metricName, metricData] of Object.entries(metrics)) {
    // Extract value and labels
    const { value, ...labels } = metricData;
    
    // Format labels as key="value" pairs
    let labelString = '';
    const formattedLabels = Object.entries(labels).map(([key, val]) => `${key}="${val}"`);
    
    if (formattedLabels.length > 0) {
      labelString = `{${formattedLabels.join(',')}}`;
    }
    
    // Add metric line - format: metric_name{label="value",...} value timestamp
    lines.push(`${metricName}${labelString} ${value} ${timestamp}`);
  }
  
  return lines.join('\n');
}

