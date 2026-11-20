// Forensic native-scan commands using mt.native() (streaming JSONL inspection).
__section__("filters", {
  title: "Filters",
  fields: {
    maxRecordBytes: { type: "int", default: 0, help: "Override MaxRecordBytes (0 = default 16MiB)" },
    stopAfter: { type: "int", default: 0, help: "Early stop after N records (0 = no early stop)" },
    path: { type: "string", default: "", help: "Specific file to inspect (empty = synthetic fixture)" },
  },
});

const REAL_FILES = [
  "/private/tmp/gmt015-persona/corpus/raw/claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl",
  "/private/tmp/gmt015-persona/corpus/raw/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl",
  "/private/tmp/gmt015-persona/corpus/raw/claude-code/agent-ae7076749749daca0.jsonl",
  "/private/tmp/gmt015-persona/corpus/raw/codex/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl",
  "/private/tmp/gmt015-persona/corpus/raw/codex/rollout-2026-10-13T13-24-04-01a1364a-f58a-7ce3-8460-a856dacc8a1a.jsonl",
  "/private/tmp/gmt015-persona/corpus/raw/codex/rollout-2026-11-10T11-59-20-01a1c666-5f02-7b92-984e-99a27afd7a94.jsonl",
  "/private/tmp/gmt015-persona/corpus/raw/pi/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl",
  "/private/tmp/gmt015-persona/corpus/raw/pi/2026-09-17T17-19-05-437Z_01a0b061-0cdd-7557-a80a-96802d3d02c5.jsonl",
  "/private/tmp/gmt015-persona/corpus/raw/pi/2026-09-29T17-08-34-793Z_01a0ee23-bd68-75c1-a81f-2c43771fc216.jsonl",
];

const SYNTH = "/private/tmp/gmt015-persona/inputs/native.jsonl";

function scanFile(path, opts) {
  const mt = require("minitrace");
  let b = mt.native().File(path);
  if (opts && opts.maxRecordBytes > 0) b = b.MaxRecordBytes(opts.maxRecordBytes);
  let records = 0;
  const typeCounts = {};
  const diagnostics = [];
  const stopped = { flag: false };
  const scan = b.ForEach((record) => {
    if (record.diagnostic) {
      diagnostics.push({
        line: record.source.line,
        byteOffset: record.source.byteOffset,
        rawBytes: record.source.rawBytes,
        code: record.diagnostic.code,
        message: record.diagnostic.message,
      });
      records++;
      return;
    }
    records++;
    const t = record.value && record.value.type ? String(record.value.type) : "(no type)";
    typeCounts[t] = (typeCounts[t] || 0) + 1;
    if (opts && opts.stopAfter > 0 && records >= opts.stopAfter) {
      stopped.flag = true;
      return false; // early termination
    }
  });
  return {
    path: path,
    sha256: scan.sha256,
    hashScope: scan.hashScope,
    bytes: scan.bytes,
    lines: scan.lines,
    malformed: scan.malformed,
    stopReason: scan.stopReason,
    recordsCounted: records,
    typeCounts: typeCounts,
    diagnosticCount: diagnostics.length,
    diagnostics: diagnostics.slice(0, 20),
    earlyStopRequested: stopped.flag,
  };
}

function streamAll(filters) {
  const out = { real: [], synthetic: null };
  for (const p of REAL_FILES) out.real.push(scanFile(p, {}));
  out.synthetic = scanFile(SYNTH, {});
  return out;
}

function earlyStop(filters) {
  return scanFile(SYNTH, { stopAfter: Number(filters.stopAfter || 3) });
}

function diagSmallLimit(filters) {
  // Force oversized-record diagnostics by lowering MaxRecordBytes to 64 bytes.
  const small = scanFile(SYNTH, { maxRecordBytes: Number(filters.maxRecordBytes || 64) });
  return small;
}

function inspectNative(filters) {
  const mt = require("minitrace");
  const path = filters.path || SYNTH;
  const evidence = mt.native().File(path)
    .MaxRecordBytes(16 * 1024 * 1024)
    .MaxIndexBytes(64 * 1024 * 1024)
    .Inspect();
  const reqs = (evidence.requests || []).map((r) => ({
    id: r.id,
    querySource: r.querySource,
    status: r.status,
    shapeHash: r.shapeHash,
    refs: r.refs,
    source: r.source,
    diagnostics: r.diagnostics,
  }));
  return {
    path: path,
    scan: {
      sha256: evidence.scan.sha256,
      hashScope: evidence.scan.hashScope,
      bytes: evidence.scan.bytes,
      lines: evidence.scan.lines,
      malformed: evidence.scan.malformed,
      stopReason: evidence.scan.stopReason,
    },
    indexBytes: evidence.indexBytes,
    requestCount: reqs.length,
    completeCount: reqs.filter((r) => r.status === "complete").length,
    incomplete: reqs.filter((r) => r.status !== "complete"),
    completeSample: reqs.filter((r) => r.status === "complete").slice(0, 3),
    blobCount: evidence.blobs ? Object.keys(evidence.blobs).length : 0,
    shapeCount: evidence.shapes ? Object.keys(evidence.shapes).length : 0,
    compactions: (evidence.compactions || []).map((c) => ({
      boundaryUuid: c.boundary && c.boundary.uuid,
      parentUuid: c.boundary && c.boundary.parentUuid,
      logicalParentUuid: c.boundary && c.boundary.logicalParentUuid,
      source: c.source,
      compactMetadata: c.compactMetadata,
      summary: c.summary,
      preceding: c.precedingMessage,
      firstFollowing: c.firstFollowingMessage,
      keys: Object.keys(c),
    })),
    diagnostics: evidence.diagnostics,
    limitations: evidence.limitations,
  };
}

__verb__("streamAll", {
  name: "stream-all",
  short: "Stream all real native transcripts plus the synthetic fixture",
  fields: { filters: { bind: "filters" } },
});
__verb__("earlyStop", {
  name: "early-stop",
  short: "Demonstrate early stop on the synthetic fixture",
  fields: { filters: { bind: "filters" } },
});
__verb__("diagSmallLimit", {
  name: "diag-small-limit",
  short: "Demonstrate oversized-record diagnostics with a small MaxRecordBytes",
  fields: { filters: { bind: "filters" } },
});
__verb__("inspectNative", {
  name: "inspect",
  short: "Inspect() native index: requests, compactions, diagnostics",
  fields: { filters: { bind: "filters" } },
});
