import { check } from 'k6';
import { browser } from 'k6/experimental/browser';

export const options = {
  scenarios: {
    ui: {
      executor: 'shared-iterations',
      options: {
        browser: {
          type: 'chromium',
        },
      },
    },
  },
};

export default async function () {
  const page = browser.newPage();

  try {
    await page.goto(`https://www.google.com`);
    console.log(page.url());
    check(page, {
      'url is correct': page.url() === `https://www.google.com`,
    });
  } finally {
    page.close();
  }
}
