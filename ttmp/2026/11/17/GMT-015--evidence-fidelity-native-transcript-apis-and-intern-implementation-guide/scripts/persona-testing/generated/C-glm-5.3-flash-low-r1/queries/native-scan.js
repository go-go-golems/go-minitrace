// native-scan.js — stream raw native transcripts via mt.native()
function scanAll() {
  const mt = require("minitrace");
  const files = [
    "/private/tmp/gmt015-persona/corpus/raw/claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/claude-code/agent-ae7076749749daca0.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/codex/rollout-2026-10-13T13-24-04-01a1364a-f58a-7ce3-8460-a856dacc8a1a.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/codex/rollout-2026-11-10T11-59-20-01a1c666-5f02-7b92-984e-99a27afd7a94.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/codex/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/pi/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/pi/2026-09-17T17-19-05-437Z_01a0b061-0cdd-7557-a80a-96802d3d02c5.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/pi/2026-09-29T17-08-34-793Z_01a0ee23-bd68-75c1-a81f-2c43771fc216.jsonl",
  ];
  const out = [];
  for (const f of files) {
    let lines = 0, malformed = 0, diag = [];
    const scan = mt.native().File(f).MaxRecordBytes(16 * 1024 * 1024).ForEach((rec) => {
      lines++;
      if (rec.diagnostic) { malformed++; diag.push({ line: rec.source.line, code: rec.diagnostic.code }); }
    });
    out.push({ path: f, sha256: scan.sha256, hashScope: scan.hashScope, bytes: scan.bytes, lines: lines, malformed: malformed, diagnostics: diag.slice(0, 5), stopReason: scan.stopReason });
  }
  return out;
}

function inspectFixture() {
  const mt = require("minitrace");
  const ev = mt.native().File("/private/tmp/gmt015-persona/inputs/native.jsonl")
    .MaxRecordBytes(16 * 1024 * 1024).MaxIndexBytes(64 * 1024 * 1024).Inspect();
  return {
    scan: { sha256: ev.scan.sha256, hashScope: ev.scan.hashScope, bytes: ev.scan.bytes, lines: ev.scan.lines, malformed: ev.scan.malformed, stopReason: ev.scan.stopReason },
    requests: ev.requests.map(r => ({ id: r.id, status: r.status, refs: r.refs, diagnostics: r.diagnostics })),
    compactions: ev.compactions.map(c => ({ boundary: c.boundary, summary: c.summary })),
    diagnostics: ev.diagnostics,
    limitations: ev.limitations,
  };
}

function earlyStop() {
  const mt = require("minitrace");
  const f = "/private/tmp/gmt015-persona/corpus/raw/claude-code/agent-ae7076749749daca0.jsonl";
  let n = 0;
  const scan = mt.native().File(f).MaxRecordBytes(16 * 1024 * 1024).ForEach((rec) => {
    n++;
    return n < 3; // stop after 3 records
  });
  return { lines: n, sha256: scan.sha256, hashScope: scan.hashScope, bytes: scan.bytes, stopReason: scan.stopReason };
}

function oversized() {
  const mt = require("minitrace");
  // tiny max record bytes to force oversized diagnostics on a real file
  const f = "/private/tmp/gmt015-persona/corpus/raw/claude-code/agent-ae7076749749daca0.jsonl";
  let diag = [], lines = 0;
  const scan = mt.native().File(f).MaxRecordBytes(64).ForEach((rec) => {
    lines++;
    if (rec.diagnostic) diag.push({ line: rec.source.line, code: rec.diagnostic.code, message: rec.diagnostic.message });
    return diag.length < 3;
  });
  return { lines, hashScope: scan.hashScope, stopReason: scan.stopReason, diagnostics: diag };
}

function realInspect() {
  const mt = require("minitrace");
  const files = [
    "/private/tmp/gmt015-persona/corpus/raw/claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/claude-code/agent-ae7076749749daca0.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/codex/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/codex/rollout-2026-11-10T11-59-20-01a1c666-5f02-7b92-984e-99a27afd7a94.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/codex/rollout-2026-10-13T13-24-04-01a1364a-f58a-7ce3-8460-a856dacc8a1a.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/pi/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/pi/2026-09-17T17-19-05-437Z_01a0b061-0cdd-7557-a80a-96802d3d02c5.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/pi/2026-09-29T17-08-34-793Z_01a0ee23-bd68-75c1-a81f-2c43771fc216.jsonl",
  ];
  const out = [];
  for (const f of files) {
    const ev = mt.native().File(f).MaxRecordBytes(16 * 1024 * 1024).MaxIndexBytes(64 * 1024 * 1024).Inspect();
    const counts = {};
    for (const r of ev.requests) counts[r.status] = (counts[r.status] || 0) + 1;
    out.push({
      path: f.replace("/private/tmp/gmt015-persona/corpus/raw/", ""),
      requests: ev.requests.length,
      statusCounts: counts,
      incomplete: ev.requests.filter(r => r.status !== "complete").slice(0, 5).map(r => ({ id: r.id, status: r.status, diagnostics: r.diagnostics })),
      compactions: ev.compactions.map(c => ({
        boundaryType: c.boundary && c.boundary.type,
        boundaryUuid: c.boundary && c.boundary.uuid,
        logicalParent: c.boundary && c.boundary.logicalParentUuid,
        boundarySource: c.boundary && c.boundary.source,
        summarySource: c.summary && c.summary.source,
      })),
      shapes: Object.keys(ev.shapes).length,
      blobs: Object.keys(ev.blobs).length,
      diagnostics: ev.diagnostics,
    });
  }
  return out;
}
