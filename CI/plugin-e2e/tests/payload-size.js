import { expect } from "@playwright/test";
import { test } from "../playwright.config";

test("payload-size", async ({ page, dashPath }) => {
  let inflatedSize = 0;
  let transferSize = 0;
  let requests = 0;

  page.on('console', msg => console.log(msg.text()));

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

  console.log({
    boot: Math.round(end - start),
    inflatedSizeMB: +(inflatedSize / 1000 / 1000).toFixed(1),
    transferSizeMB: +(transferSize / 1000 / 1000).toFixed(1),
    requests: requests,
    usedJSHeapSize: +(usedJSHeapSize / 1000 / 1000).toFixed(1),
  });

  // if we don't remove the listener the "test" will error.
  page.removeListener("response", addSize);

  client.detach();
}).tag('@performance');
