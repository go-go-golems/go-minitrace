const RAW = "/private/tmp/gmt015-persona/corpus/raw";
const SYNTH = "/private/tmp/gmt015-persona/inputs/native.jsonl";

__section__("options", {
  title: "Options",
  fields: {
    path: { type: "string", help: "Native JSONL path" },
  },
});

function scanAll(options) {
  const mt = require("minitrace");
  const files = [
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
  return files.map((path) => {
    const seen = { total: 0, types: {} };
    const scan = mt.native().File(path).ForEach((record) => {
      seen.total++;
      const t = record.diagnostic ? "MALFORMED" : String(record.value && record.value.type || "(none)");
      seen.types[t] = (seen.types[t] || 0) + 1;
    });
    return {
      path,
      sha256: scan.sha256,
      hashScope: scan.hashScope,
      bytes: scan.bytes,
      lines: scan.lines,
      malformed: scan.malformed,
      stopReason: scan.stopReason,
      records: seen.total,
      types: seen.types,
    };
  });
}

function earlyStop(options) {
  const mt = require("minitrace");
  let n = 0;
  const scan = mt.native().File(SYNTH).ForEach((record) => {
    n++;
    if (n >= 3) return false;
  });
  return { calls: n, scan };
}

function inspectSynthetic(options) {
  const mt = require("minitrace");
  const ev = mt.native().File(SYNTH).Inspect();
  return {
    scan: ev.scan,
    indexBytes: ev.indexBytes,
    limitations: ev.limitations,
    diagnostics: ev.diagnostics,
    requests: ev.requests,
    compactions: ev.compactions,
    shapeKeys: Object.keys(ev.shapes || {}),
    blobKeys: Object.keys(ev.blobs || {}),
  };
}

__verb__("scanAll", { name: "scan-all", short: "Stream all native transcripts", fields: { options: { bind: "options" } } });
__verb__("earlyStop", { name: "early-stop", short: "Demonstrate early stop hash scope", fields: { options: { bind: "options" } } });
__verb__("inspectSynthetic", { name: "inspect-synthetic", short: "Inspect synthetic native fixture", fields: { options: { bind: "options" } } });

function inspectReal(options) {
  const mt = require("minitrace");
  const path = options.path;
  const ev = mt.native().File(path).Inspect();
  const reqs = ev.requests.map((r) => ({
    id: r.id,
    status: r.status,
    line: r.source && r.source.line,
    byteOffset: r.source && r.source.byteOffset,
    shapeHash: r.shapeHash,
    base: r.refs && r.refs.base || "",
    refs: r.refs && r.refs.tail || [],
    diagnostics: r.diagnostics || [],
  }));
  return {
    path,
    scan: ev.scan,
    indexBytes: ev.indexBytes,
    counts: {
      requests: reqs.length,
      complete: reqs.filter((r) => r.status === "complete").length,
      incomplete: reqs.filter((r) => r.status !== "complete").length,
      byStatus: reqs.reduce((m, r) => { m[r.status] = (m[r.status] || 0) + 1; return m; }, {}),
    },
    complete: reqs.filter((r) => r.status === "complete"),
    incomplete: reqs.filter((r) => r.status !== "complete"),
    compactions: ev.compactions,
    diagnostics: ev.diagnostics,
    limitations: ev.limitations,
  };
}

function diagnostics(options) {
  const mt = require("minitrace");
  const out = [];
  const scan = mt.native().File(SYNTH).MaxRecordBytes(64).ForEach((record) => {
    out.push({
      line: record.source.line,
      byteOffset: record.source.byteOffset,
      rawBytes: record.source.rawBytes,
      diagnostic: record.diagnostic || null,
      type: record.value && record.value.type || null,
    });
  });
  return { scan, records: out };
}

__verb__("inspectReal", { name: "inspect-real", short: "Inspect a real Claude native source", fields: { options: { bind: "options" } } });
__verb__("diagnostics", { name: "diagnostics", short: "Malformed and oversized diagnostics", fields: { options: { bind: "options" } } });
