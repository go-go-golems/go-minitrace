#!/usr/bin/env node

const BASE_URL = process.env.BASE_URL || "http://127.0.0.1:18081";
const SESSION_ID = process.env.SESSION_ID || "019d0295-d06b-7033-b154-a991a94672b6";
const ITERATIONS = Number(process.env.ITERATIONS || 5);

function mean(values) {
  return values.reduce((a, b) => a + b, 0) / Math.max(values.length, 1);
}

async function measureJson(path) {
  const url = `${BASE_URL}${path}`;
  const t0 = performance.now();
  const res = await fetch(url);
  const text = await res.text();
  const t1 = performance.now();
  let json = null;
  try {
    json = JSON.parse(text);
  } catch {
    json = null;
  }
  return {
    url,
    status: res.status,
    durationMs: Math.round(t1 - t0),
    bytes: Buffer.byteLength(text),
    json,
  };
}

const summaryRuns = [];
const blocksRuns = [];
const detailRuns = [];

for (let i = 0; i < ITERATIONS; i += 1) {
  summaryRuns.push(await measureJson(`/api/sessions/${SESSION_ID}/summary`));
  blocksRuns.push(await measureJson(`/api/sessions/${SESSION_ID}/blocks`));
  detailRuns.push(await measureJson(`/api/sessions/${SESSION_ID}`));
}

const lastSummary = summaryRuns.at(-1);
const lastBlocks = blocksRuns.at(-1);
const lastDetail = detailRuns.at(-1);

const summary = {
  baseUrl: BASE_URL,
  sessionId: SESSION_ID,
  iterations: ITERATIONS,
  summaryEndpoint: {
    meanDurationMs: Math.round(mean(summaryRuns.map((r) => r.durationMs))),
    meanBytes: Math.round(mean(summaryRuns.map((r) => r.bytes))),
    last: {
      status: lastSummary?.status,
      durationMs: lastSummary?.durationMs,
      bytes: lastSummary?.bytes,
      hasBlocksField: Object.prototype.hasOwnProperty.call(lastSummary?.json ?? {}, "blocks"),
      title: lastSummary?.json?.title ?? null,
    },
    runs: summaryRuns.map((r) => ({ status: r.status, durationMs: r.durationMs, bytes: r.bytes })),
  },
  blocksEndpoint: {
    meanDurationMs: Math.round(mean(blocksRuns.map((r) => r.durationMs))),
    meanBytes: Math.round(mean(blocksRuns.map((r) => r.bytes))),
    last: {
      status: lastBlocks?.status,
      durationMs: lastBlocks?.durationMs,
      bytes: lastBlocks?.bytes,
      blockCount: Array.isArray(lastBlocks?.json) ? lastBlocks.json.length : null,
    },
    runs: blocksRuns.map((r) => ({ status: r.status, durationMs: r.durationMs, bytes: r.bytes })),
  },
  fullDetailEndpoint: {
    meanDurationMs: Math.round(mean(detailRuns.map((r) => r.durationMs))),
    meanBytes: Math.round(mean(detailRuns.map((r) => r.bytes))),
    last: {
      status: lastDetail?.status,
      durationMs: lastDetail?.durationMs,
      bytes: lastDetail?.bytes,
      blockCount: Array.isArray(lastDetail?.json?.blocks) ? lastDetail.json.blocks.length : null,
    },
    runs: detailRuns.map((r) => ({ status: r.status, durationMs: r.durationMs, bytes: r.bytes })),
  },
};

console.log(JSON.stringify(summary, null, 2));
