// Native JSONL evidence commands using mt.native()
var RAW = "/private/tmp/gmt015-persona/corpus/raw";
var FILES = [
  RAW + "/claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl",
  RAW + "/claude-code/agent-ae7076749749daca0.jsonl",
  RAW + "/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl",
  RAW + "/codex/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl",
  RAW + "/codex/rollout-2026-10-13T13-24-04-01a1364a-f58a-7ce3-8460-a856dacc8a1a.jsonl",
  RAW + "/codex/rollout-2026-11-10T11-59-20-01a1c666-5f02-7b92-984e-99a27afd7a94.jsonl",
  RAW + "/pi/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl",
  RAW + "/pi/2026-09-17T17-19-05-437Z_01a0b061-0cdd-7557-a80a-96802d3d02c5.jsonl",
  RAW + "/pi/2026-09-29T17-08-34-793Z_01a0ee23-bd68-75c1-a81f-2c43771fc216.jsonl",
];
var FIXTURE = "/private/tmp/gmt015-persona/inputs/native.jsonl";

function scanOne(path, opts) {
  const mt = require("minitrace");
  let b = mt.native().File(path);
  if (opts && opts.maxRecordBytes) b = b.MaxRecordBytes(opts.maxRecordBytes);
  let count = 0, apiRequests = 0, diags = [], types = {};
  const scan = b.ForEach(function (record) {
    count++;
    if (record.diagnostic) { diags.push({ line: record.source.line, byteOffset: record.source.byteOffset, code: record.diagnostic.code, message: record.diagnostic.message }); return; }
    const t = record.value && record.value.type;
    types[t === undefined ? "none" : t] = (types[t === undefined ? "none" : t] || 0) + 1;
    if (t === "api-request") apiRequests++;
  });
  return { path: path, scan: scan, recordCount: count, apiRequestCount: apiRequests, typeCounts: types, diagnostics: diags };
}

function scanAll() {
  return FILES.map(function (p) { return scanOne(p, null); });
}

__verb__("nativeScanAll", {
  name: "native-scan-all",
  short: "Stream all real native transcripts with mt.native().ForEach",
});

function nativeScanAll() { return scanAll(); }

__verb__("nativeInspectFixture", {
  name: "native-inspect-fixture",
  short: "Inspect() the synthetic native fixture",
});

function nativeInspectFixture() {
  const mt = require("minitrace");
  const ev = mt.native().File(FIXTURE).Inspect();
  return {
    scan: ev.scan,
    limitations: ev.limitations,
    requestCount: ev.requests.length,
    requests: ev.requests.map(function (r) { return { id: r.id, querySource: r.querySource, source: r.source, status: r.status, diagnostics: r.diagnostics, refs: r.refs }; }),
    compactions: ev.compactions,
    diagnostics: ev.diagnostics,
    indexBytes: ev.indexBytes,
    blobCount: ev.blobs ? Object.keys(ev.blobs).length : null,
    shapeCount: ev.shapes ? Object.keys(ev.shapes).length : null,
  };
}

__verb__("nativeInspectClaude", {
  name: "native-inspect-claude",
  short: "Inspect() a real Claude source containing api-request records",
});

function nativeInspectClaude() {
  const mt = require("minitrace");
  // find a claude file with api-request records first
  const found = scanAll().filter(function (s) { return s.apiRequestCount > 0 && s.path.indexOf("claude") >= 0; });
  if (!found.length) return { error: "no claude api-request file" };
  const path = found[0].path;
  const ev = mt.native().File(path).Inspect();
  return {
    path: path,
    scan: ev.scan,
    requestCount: ev.requests.length,
    requests: ev.requests.map(function (r) { return { id: r.id, source: r.source, status: r.status, diagnostics: r.diagnostics, shapeHash: r.shapeHash }; }),
    compactions: ev.compactions,
    diagnostics: ev.diagnostics,
    limitations: ev.limitations,
    indexBytes: ev.indexBytes,
  };
}

__verb__("nativeEarlyStop", {
  name: "native-early-stop",
  short: "Demonstrate early stop: consume only first 5 records, show hashScope consumed_prefix",
});

function nativeEarlyStop() {
  const mt = require("minitrace");
  let n = 0;
  const scan = mt.native().File(FILES[0]).ForEach(function () { n++; if (n >= 5) return false; });
  return { stoppedAfter: n, hashScope: scan.hashScope, sha256: scan.sha256, bytes: scan.bytes, lines: scan.lines, stopReason: scan.stopReason, malformed: scan.malformed };
}

__verb__("nativeMalformed", {
  name: "native-malformed",
  short: "Demonstrate malformed/oversized-record diagnostics with a tiny MaxRecordBytes",
});

function nativeMalformed() {
  return scanOne(FILES[0], { maxRecordBytes: 400 });
}
