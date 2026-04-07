import { chromium } from "../../../../../../web/node_modules/playwright/index.mjs";

const baseURL = process.env.BASE_URL || "http://127.0.0.1:5173";
const sessionTitle = "Here's a plugin experiment in the browser";
const toolCallId = "call_Y70XEopD3Ef1mGctwTXG2CEq";

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 1600, height: 1200 } });

try {
  console.log(`Navigate: ${baseURL}`);
  await page.goto(baseURL, { waitUntil: "networkidle" });

  await page.waitForSelector('[data-widget="session-browser"]', { timeout: 15000 });
  await page.getByText(sessionTitle, { exact: false }).click();

  await page.waitForSelector('[data-widget="transcript-viewer"]', { timeout: 15000 });
  console.log("Opened transcript viewer");

  // Turn scoped annotation: card click should switch to transcript and reveal turn 0.
  await page.getByRole("tab", { name: /Annotations/ }).click();
  await page.getByText("Turn scoped smoke", { exact: false }).click();
  await page.waitForSelector('[data-turn-idx="0"]', { timeout: 10000 });
  const turnVisible = await page.locator('[data-turn-idx="0"]').isVisible();
  assert(turnVisible, "turn target did not become visible after annotation click");
  console.log("PASS: clicking turn-scoped annotation reveals turn target");

  // Tool scoped annotation: card click should reveal tool call row.
  await page.getByRole("tab", { name: /Annotations/ }).click();
  await page.getByText("Tool scoped smoke", { exact: false }).click();
  await page.waitForSelector(`[data-tool-call-id="${toolCallId}"]`, { timeout: 10000 });
  const toolVisible = await page.locator(`[data-tool-call-id="${toolCallId}"]`).isVisible();
  assert(toolVisible, "tool-call target did not become visible after annotation click");
  console.log("PASS: clicking tool-call-scoped annotation reveals tool target");

  // In-context turn annotation button should open annotation form with turn scope.
  await page.locator('[data-turn-idx="0"]').getByRole('button', { name: 'Annotate' }).click();
  await page.waitForSelector('[data-widget="annotation-panel"]', { timeout: 10000 });
  const scopeText = await page.locator('[data-widget="annotation-panel"]').textContent();
  assert(scopeText?.includes('Scope: turn') && scopeText?.includes('target 0'), 'turn annotate action did not prefill turn scope');
  console.log('PASS: turn Annotate action prefills turn scope');

  // Return to transcript, then tool-call annotate button should open tool_call scope.
  await page.getByRole('tab', { name: /Transcript/ }).click();
  await page.locator(`[data-tool-call-id="${toolCallId}"]`).getByRole('button', { name: 'Annotate' }).click();
  await page.waitForSelector('[data-widget="annotation-panel"]', { timeout: 10000 });
  const scopeText2 = await page.locator('[data-widget="annotation-panel"]').textContent();
  assert(scopeText2?.includes('Scope: tool_call') && scopeText2?.includes(toolCallId), 'tool-call annotate action did not prefill tool_call scope');
  console.log('PASS: tool-call Annotate action prefills tool_call scope');

  await page.screenshot({ path: 'tmp/ui-smoke/ui-smoke-annotation-navigation.png', fullPage: true });
  console.log('Saved screenshot: tmp/ui-smoke/ui-smoke-annotation-navigation.png');
  console.log('ALL UI SMOKE TESTS PASSED');
} finally {
  await browser.close();
}
