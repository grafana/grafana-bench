import { expect } from "@playwright/test";
import { test } from "../playwright.config";
import  fs from "fs";

test("payload-size",{ tag: '@performance' }, async ({ page, dashPath }) => {
  let inflatedSize = 0;
  let transferSize = 0;
  let requests = 0;

  //page.on('console', msg => console.log(msg.text()));

  const addSize = async (response) => {
    if (response.status() === 200) {
      let body = await response.body();
      inflatedSize += body.length;

      const sizes = await response.request().sizes();

      transferSize += sizes.responseBodySize + sizes.responseHeadersSize;

      requests++;
    }
  };

  page.on("response", addSize);

  let start = performance.now();

  await page.goto(dashPath);

  let el = page.getByTestId("header-container");
  await el.waitFor();

  // weird but random expect() is required so this whole thing doesn't go tits up with
  // "Error: page.goto: net::ERR_ABORTED; maybe frame was detached?"
  await expect(el).toBeVisible();

  let end = performance.now();

  let client = await page.context().newCDPSession(page);
  // await page.evaluate(() => window.gc());
  await client.send('HeapProfiler.collectGarbage');
  let usedJSHeapSize = (await client.send("Runtime.getHeapUsage")).usedSize;

  //console.log({
  //  boot: Math.round(end - start),
  //  inflatedSizeMB: +(inflatedSize / 1000 / 1000).toFixed(1),
  //  transferSizeMB: +(transferSize / 1000 / 1000).toFixed(1),
  //  requests: requests,
  //  usedJSHeapSize: +(usedJSHeapSize / 1000 / 1000).toFixed(1),
  //});

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

  // Write json data to file
  const textExpositionData = convertToPrometheusFormat(metricsWithLabels);
  console.log(textExpositionData);
  fs.writeFileSync('/tmp/asset-metrics.txt', textExpositionData);

  // if we don't remove the listener the "test" will error.
  page.removeListener("response", addSize);
  client.detach();
  page.close()
});


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

