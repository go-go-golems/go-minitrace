#!/usr/bin/env node
import { chromium } from "../../../../../../web/node_modules/playwright/index.mjs";

const BASE_URL = process.env.BASE_URL || "http://127.0.0.1:5174";
const API_URL = process.env.API_URL || "http://127.0.0.1:8080";
const SESSION_ID = process.env.SESSION_ID || "019d0295-d06b-7033-b154-a991a94672b6";
const ITERATIONS = Number(process.env.ITERATIONS || 3);

function mean(values) {
  return values.reduce((a, b) => a + b, 0) / Math.max(values.length, 1);
}

async function fetchJson(url) {
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`fetch failed ${res.status}: ${url}`);
  }
  return await res.json();
}

async function measureSessionBrowser(page) {
  const t0 = Date.now();
  await page.goto(`${BASE_URL}/sessions`, { waitUntil: "domcontentloaded" });
  await page.waitForSelector('[data-widget="session-browser"]', { timeout: 15000 });
  await page.waitForSelector('tbody tr', { timeout: 15000 });
  const t1 = Date.now();
  const stats = await page.evaluate(() => ({
    rows: document.querySelectorAll('tbody tr').length,
    nodes: document.querySelectorAll('*').length,
  }));
  return { loadMs: t1 - t0, ...stats };
}

async function measureTranscript(page) {
  const t0 = Date.now();
  await page.goto(`${BASE_URL}/sessions/${SESSION_ID}?tab=transcript`, {
    waitUntil: "domcontentloaded",
  });
  await page.waitForSelector('[data-widget="transcript-viewer"]', { timeout: 15000 });
  await page.waitForSelector('[data-part="block"]', { timeout: 15000 });
  const t1 = Date.now();
  const initial = await page.evaluate(() => ({
    blocks: document.querySelectorAll('[data-part="block"]').length,
    turns: document.querySelectorAll('[data-turn-idx]').length,
    toolCalls: document.querySelectorAll('[data-tool-call-id]').length,
    nodes: document.querySelectorAll('*').length,
  }));

  await page.getByRole('tab', { name: 'Annotations' }).click();
  await page.waitForSelector('[data-widget="annotation-panel"]', { timeout: 10000 });
  const t2 = Date.now();

  await page.getByRole('tab', { name: /Transcript/ }).click();
  await page.waitForFunction(() => {
    const el = document.querySelector('[data-part="block"]');
    return !!el;
  }, { timeout: 10000 });
  const t3 = Date.now();

  return {
    initialLoadMs: t1 - t0,
    toAnnotationsMs: t2 - t1,
    backToTranscriptMs: t3 - t2,
    ...initial,
  };
}

async function measureQueryPage(page) {
  const t0 = Date.now();
  await page.goto(`${BASE_URL}/query`, { waitUntil: "domcontentloaded" });
  await page.waitForSelector('[data-widget="query-editor"]', { timeout: 15000 });
  const t1 = Date.now();
  const stats = await page.evaluate(() => ({
    nodes: document.querySelectorAll('*').length,
  }));
  return { loadMs: t1 - t0, ...stats };
}

const session = await fetchJson(`${API_URL}/api/sessions/${SESSION_ID}`);
const sessionStats = {
  id: session.id,
  title: String(session.title || '').slice(0, 120),
  blocks: session.blocks.length,
  turns: session.blocks.reduce((a, b) => a + b.turns.length, 0),
  toolCalls: session.blocks.reduce(
    (a, b) => a + b.turns.reduce((x, t) => x + t.tool_calls_in_turn.length, 0),
    0,
  ),
  model: session.environment.model,
};

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } });

try {
  const sessionBrowserRuns = [];
  const transcriptRuns = [];
  const queryRuns = [];

  for (let i = 0; i < ITERATIONS; i += 1) {
    sessionBrowserRuns.push(await measureSessionBrowser(page));
    transcriptRuns.push(await measureTranscript(page));
    queryRuns.push(await measureQueryPage(page));
  }

  const summary = {
    baseUrl: BASE_URL,
    apiUrl: API_URL,
    iterations: ITERATIONS,
    session: sessionStats,
    sessionBrowser: {
      meanLoadMs: Math.round(mean(sessionBrowserRuns.map((r) => r.loadMs))),
      last: sessionBrowserRuns.at(-1),
      runs: sessionBrowserRuns,
    },
    transcript: {
      meanInitialLoadMs: Math.round(mean(transcriptRuns.map((r) => r.initialLoadMs))),
      meanToAnnotationsMs: Math.round(mean(transcriptRuns.map((r) => r.toAnnotationsMs))),
      meanBackToTranscriptMs: Math.round(mean(transcriptRuns.map((r) => r.backToTranscriptMs))),
      last: transcriptRuns.at(-1),
      runs: transcriptRuns,
    },
    queryPage: {
      meanLoadMs: Math.round(mean(queryRuns.map((r) => r.loadMs))),
      last: queryRuns.at(-1),
      runs: queryRuns,
    },
  };

  console.log(JSON.stringify(summary, null, 2));
} finally {
  await browser.close();
}
