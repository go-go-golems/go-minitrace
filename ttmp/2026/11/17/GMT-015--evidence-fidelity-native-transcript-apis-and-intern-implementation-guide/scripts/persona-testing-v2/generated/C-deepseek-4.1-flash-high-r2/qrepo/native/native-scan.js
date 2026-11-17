// Scanner-first JS command: bounded native evidence scan over the nine real
// raw sources under corpus/raw plus the synthetic fixture native.jsonl.
// Real counting uses mt.native().ForEach (streaming, no index) for record
// types; synthetic request-graph completeness uses mt.native().Inspect().
__section__("scan", {
  title: "Scan",
  fields: {
    mode: { type: "string", default: "all", help: "all|real|synthetic|fidelity" },
  },
});

const RAW_BASE = "/private/tmp/gmt015-persona-v2-matrix/corpus/raw";
const SYNTH = "/private/tmp/gmt015-persona-v2-matrix/fixtures/native.jsonl";
const REAL_REL = [
  "claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl",
  "claude-code/agent-ae7076749749daca0.jsonl",
  "claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl",
  "codex/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl",
  "codex/rollout-2026-10-13T13-24-04-01a1364a-f58a-7ce3-8460-a856dacc8a1a.jsonl",
  "codex/rollout-2026-11-10T11-59-20-01a1c666-5f02-7b92-984e-99a27afd7a94.jsonl",
  "pi/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl",
  "pi/2026-09-17T17-19-05-437Z_01a0b061-0cdd-7557-a80a-96802d3d02c5.jsonl",
  "pi/2026-09-29T17-08-34-793Z_01a0ee23-bd68-75c1-a81f-2c43771fc216.jsonl",
];

function shapeOf(record) {
  const v = record.value;
  if (v === null) return "null";
  if (Array.isArray(v)) return "array";
  if (typeof v !== "object") return typeof v;
  if (typeof v.type === "string") return "type=" + v.type;
  return "object(no type)";
}

function scanReal(filters) {
  const mt = require("minitrace");
  const out = [];
  for (const rel of REAL_REL) {
    const path = RAW_BASE + "/" + rel;
    const counts = {};
    let apiRequests = 0;
    let malformed = 0;
    let recordCount = 0;
    const diagnostics = [];
    const scan = mt.native().File(path)
      .MaxRecordBytes(16 * 1024 * 1024)
      .ForEach(function (record) {
        if (record.diagnostic) {
          malformed++;
          if (diagnostics.length < 5) {
            diagnostics.push({ line: record.source.line, code: record.diagnostic.code, message: record.diagnostic.message, rawBytes: record.source.rawBytes });
          }
          return true;
        }
        recordCount++;
        const k = shapeOf(record);
        counts[k] = (counts[k] || 0) + 1;
        if (record.value && record.value.type === "api-request") apiRequests++;
        return true;
      });
    out.push({
      path_relative_to_raw_base: rel,
      absolute_path: path,
      sha256: scan.sha256,
      bytes: scan.bytes,
      lines: scan.lines,
      hashScope: scan.hashScope,
      stopReason: scan.stopReason,
      malformed_report_count: scan.malformed,
      malformed_observed: malformed,
      parsed_record_count: recordCount,
      api_request_records: apiRequests,
      top_record_shapes: counts,
      malformed_examples: diagnostics,
    });
  }
  return out;
}

function inspectSynthetic(filters) {
  const mt = require("minitrace");
  const evidence = mt.native().File(SYNTH).Inspect();
  const requests = evidence.requests.map(function (r) {
    return {
      id: r.id,
      status: r.status,
      querySource: r.querySource,
      line: r.source ? r.source.line : null,
      shapeHash: r.shapeHash || null,
      refs: r.refs || null,
      diagnostics: (r.diagnostics || []).map(function (d) { return { code: d.code, message: d.message }; }),
    };
  });
  const complete = requests.filter(function (r) { return r.status === "complete"; }).length;
  const incomplete = requests.filter(function (r) { return r.status !== "complete"; }).length;
  const compactions = (evidence.compactions || []).map(function (c) {
    return {
      boundaryLine: c.boundary.source ? c.boundary.source.line : null,
      boundaryUuid: c.boundary.uuid || null,
      boundaryType: c.boundary.type || null,
      logicalParentUuid: c.logicalParentUuid === undefined ? null : c.logicalParentUuid,
      metadata: c.metadata === undefined ? null : c.metadata,
      summaries: (c.summaries || []).map(function (s) { return { uuid: s.uuid, line: s.source ? s.source.line : null }; }),
      precedingUuid: c.beforeMessage ? c.beforeMessage.uuid : null,
      firstFollowingUuid: c.firstFollowingMessage ? c.firstFollowingMessage.uuid : null,
      firstFollowingAssistantUuid: c.firstFollowingAssistant ? c.firstFollowingAssistant.uuid : null,
    };
  });
  return {
    scan: {
      sha256: evidence.scan.sha256,
      hashScope: evidence.scan.hashScope,
      bytes: evidence.scan.bytes,
      lines: evidence.scan.lines,
      malformed: evidence.scan.malformed,
      stopReason: evidence.scan.stopReason,
    },
    indexBytes: evidence.indexBytes,
    limitations: evidence.limitations,
    requests: requests,
    requestCounts: { requests: requests.length, complete: complete, incomplete: incomplete },
    compactions: compactions,
    diagnostics: (evidence.diagnostics || []).map(function (d) {
      return { code: d.code, message: d.message, line: d.source ? d.source.line : null };
    }),
    shapes: Object.keys(evidence.shapes || {}),
    blobs: Object.keys(evidence.blobs || {}),
  };
}

function realScan(filters) {
  return scanReal(filters);
}

function syntheticScan(filters) {
  return inspectSynthetic(filters);
}

// Optional fidelity probes: (a) early stop => hashScope consumed_prefix,
// (b) small MaxRecordBytes => oversized diagnostics without dropping records.
function fidelityProbes(filters) {
  const mt = require("minitrace");
  const path = RAW_BASE + "/claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl";

  let seen = 0;
  const earlyStop = mt.native().File(path).ForEach(function (record) {
    seen++;
    return seen < 3; // false on the 3rd record terminates iteration
  });

  const diagCounts = {};
  const diagExamples = [];
  let parsed = 0;
  const oversized = mt.native().File(path)
    .MaxRecordBytes(1024)
    .ForEach(function (record) {
      if (record.diagnostic) {
        diagCounts[record.diagnostic.code] = (diagCounts[record.diagnostic.code] || 0) + 1;
        if (diagExamples.length < 3) {
          diagExamples.push({ line: record.source.line, code: record.diagnostic.code, rawBytes: record.source.rawBytes, message: record.diagnostic.message });
        }
        return true;
      }
      parsed++;
      return true;
    });

  return [
    {
      probe: "early_stop_after_3_records",
      file: path,
      records_seen: seen,
      sha256: earlyStop.sha256,
      hashScope: earlyStop.hashScope,
      bytes: earlyStop.bytes,
      lines: earlyStop.lines,
      stopReason: earlyStop.stopReason,
      malformed: earlyStop.malformed,
    },
    {
      probe: "max_record_bytes_1024",
      file: path,
      parsed_records: parsed,
      diagnostic_counts: diagCounts,
      diagnostic_examples: diagExamples,
      sha256: oversized.sha256,
      hashScope: oversized.hashScope,
      bytes: oversized.bytes,
      lines: oversized.lines,
      stopReason: oversized.stopReason,
    },
  ];
}

__verb__("fidelityProbes", {
  name: "fidelity-probes",
  short: "Probe early-stop hash scope and oversized-record diagnostics on one real source",
  fields: { filters: { bind: "scan" } },
});

__verb__("realScan", {
  name: "real-scan",
  short: "Stream all nine real raw native sources and report hash/size/line/api-request counts",
  fields: { filters: { bind: "scan" } },
});

__verb__("syntheticScan", {
  name: "synthetic-scan",
  short: "Inspect the synthetic native.jsonl fixture request graphs and compaction boundary",
  fields: { filters: { bind: "scan" } },
});
