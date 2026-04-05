import { chromium } from "../../../../../../web/node_modules/playwright/index.mjs";

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 1600, height: 1200 } });
const sid = process.env.SESSION_ID || "019d0295-d06b-7033-b154-a991a94672b6";
const base = process.env.BASE_URL || "http://127.0.0.1:5173";

function assert(cond, msg) {
  if (!cond) throw new Error(msg);
}

try {
  await page.goto(`${base}/sessions/${sid}`, { waitUntil: "networkidle" });
  await page.waitForSelector('[data-widget="transcript-viewer"]', { timeout: 15000 });

  // 1) Turn annotate opens inline composer and does not switch tabs.
  await page.locator('[data-turn-idx="0"]').getByRole('button', { name: 'Annotate' }).click();
  const selectedTab = await page.locator('[role="tab"][aria-selected="true"]').textContent();
  assert(selectedTab?.includes('Transcript'), 'Annotate should keep Transcript tab selected');
  const transcriptText = await page.locator('[data-widget="transcript-viewer"]').textContent();
  assert(transcriptText?.includes('Add turn annotation'), 'Inline transcript composer did not appear');
  assert(page.url().includes('composeType=turn') && page.url().includes('composeTarget=0'), 'URL missing inline composer state');
  console.log('PASS: inline composer opens in transcript and URL stores compose target');

  // 2) Clicking inline chip opens annotations tab and stores annotation selection in URL.
  await page.locator('[data-turn-idx="0"]').locator('.MuiChip-root').filter({ hasText: 'observation' }).first().click();
  await page.waitForSelector('[data-widget="annotation-panel"]', { timeout: 10000 });
  const selectedTab2 = await page.locator('[role="tab"][aria-selected="true"]').textContent();
  assert(selectedTab2?.includes('Annotations'), 'Inline chip should switch to Annotations tab');
  assert(page.url().includes('tab=annotations') && page.url().includes('annotation='), 'URL missing selected annotation state');
  console.log('PASS: clicking inline chip switches to annotations tab and URL stores selected annotation');

  // 3) Clicking annotation card returns to transcript and stores focus target in URL.
  await page.locator('[data-annotation-id]').first().click();
  const selectedTab3 = await page.locator('[role="tab"][aria-selected="true"]').textContent();
  assert(selectedTab3?.includes('Transcript'), 'Clicking annotation card should switch to Transcript tab');
  assert(page.url().includes('focusType=turn') && page.url().includes('focusId='), 'URL missing focused target state');
  console.log('PASS: clicking annotation card switches to transcript and URL stores focus target');

  await page.screenshot({ path: 'tmp/ui-smoke/ui-workflow-live-stack-smoke.png', fullPage: true });
  console.log('Saved screenshot: tmp/ui-smoke/ui-workflow-live-stack-smoke.png');
  console.log('ALL LIVE STACK UX SMOKE TESTS PASSED');
} finally {
  await browser.close();
}
