// Forensic native-evidence scans. Read-only: mt.native() streams bytes and
// never evaluates transcript content. No DB, no cache, no writes.
__section__("options", {
  title: "Scan options",
  fields: {
    note: { type: "string", default: "read-only native scans", help: "Free-form label; no effect" },
  },
});

const RAW = "/private/tmp/gmt015-persona/corpus/raw";
const SYNTH = "/private/tmp/gmt015-persona/inputs/native.jsonl";

const REAL_FILES = [
  RAW + "/claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl",
  RAW + "/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl",
  RAW + "/claude-code/agent-ae7076749749daca0.jsonl",
  RAW + "/codex/rollout-2026-10-13T13-24-04-01a1364a-f58a-7ce3-8460-a856dacc8a1a.jsonl",
  RAW + "/codex/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl",
  RAW + "/codex/rollout-2026-11-10T11-59-20-01a1c666-5f02-7b92-984e-99a27afd7a94.jsonl",
  RAW + "/pi/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl",
  RAW + "/pi/2026-09-17T17-19-05-437Z_01a0b061-0cdd-7557-a80a-96802d3d02c5.jsonl",
  RAW + "/pi/2026-09-29T17-08-34-793Z_01a0ee23-bd68-75c1-a81f-2c43771fc216.jsonl",
];

function scanOne(mt, path) {
  const typeCounts = {};
  const requestish = {};
  let records = 0;
  let malformed = 0;
  let firstMalformed = null;
  const stats = mt.native().File(path).ForEach((rec) => {
    records++;
    if (rec.diagnostic) {
      malformed++;
      if (!firstMalformed) {
        firstMalformed = { line: rec.source.line, diagnostic: rec.diagnostic };
      }
      return true;
    }
    const t = rec.value && rec.value.type ? String(rec.value.type) : "(none)";
    typeCounts[t] = (typeCounts[t] || 0) + 1;
    if (t.indexOf("api-request") === 0) {
      requestish[t] = (requestish[t] || 0) + 1;
    }
    return true;
  });
  return {
    path: path,
    sha256: stats.sha256,
    hashScope: stats.hashScope,
    bytes: stats.bytes,
    lines: stats.lines,
    loopCallbacks: records,
    malformed: stats.malformed,
    malformedInCallback: malformed,
    stopReason: stats.stopReason,
    firstMalformed: firstMalformed,
    apiRequestTypeCounts: requestish,
    distinctTypes: Object.keys(typeCounts).length,
    typeCountsTop: typeCounts,
  };
}

function scanAll() {
  const mt = require("minitrace");
  const out = [];
  for (let i = 0; i < REAL_FILES.length; i++) {
    out.push(scanOne(mt, REAL_FILES[i]));
  }
  return out;
}

function earlyStop() {
  const mt = require("minitrace");
  let seen = 0;
  const stats = mt.native()
    .File(REAL_FILES[0])
    .ForEach((rec) => {
      seen++;
      return seen < 5; // 5th callback returns false => consumed prefix only
    });
  return [{
    path: REAL_FILES[0],
    wholeFileBytes: 3779472,
    callbacks: seen,
    sha256: stats.sha256,
    hashScope: stats.hashScope,
    bytes: stats.bytes,
    lines: stats.lines,
    stopReason: stats.stopReason,
    note: "prefix hash is not the file hash; a stopped scan cannot certify whole-source integrity",
  }];
}

function diagnosticsDemo() {
  const mt = require("minitrace");
  const found = [];
  const stats = mt.native()
    .File(SYNTH)
    .MaxRecordBytes(32) // force "oversized" for every line longer than 32 bytes
    .ForEach((rec) => {
      if (rec.diagnostic) {
        found.push({
          line: rec.source.line,
          byteOffset: rec.source.byteOffset,
          rawBytes: rec.source.rawBytes,
          code: rec.diagnostic.code,
          message: rec.diagnostic.message,
        });
      }
      return true;
    });
  return [{
    path: SYNTH,
    maxRecordBytes: 32,
    sha256: stats.sha256,
    hashScope: stats.hashScope,
    bytes: stats.bytes,
    lines: stats.lines,
    malformed: stats.malformed,
    stopReason: stats.stopReason,
    diagnostics: found,
  }];
}

function malformedOnly() {
  const mt = require("minitrace");
  const found = [];
  const stats = mt.native().File(SYNTH).ForEach((rec) => {
    if (rec.diagnostic) {
      found.push({
        line: rec.source.line,
        byteOffset: rec.source.byteOffset,
        rawBytes: rec.source.rawBytes,
        code: rec.diagnostic.code,
        message: rec.diagnostic.message,
      });
    }
    return true;
  });
  return [{
    path: SYNTH,
    sha256: stats.sha256,
    hashScope: stats.hashScope,
    bytes: stats.bytes,
    lines: stats.lines,
    malformed: stats.malformed,
    stopReason: stats.stopReason,
    diagnostics: found,
  }];
}

function synthInspect() {
  const mt = require("minitrace");
  const rep = mt.native().File(SYNTH).Inspect();
  return [{
    path: SYNTH,
    scan: rep.scan,
    requests: rep.requests,
    compactions: rep.compactions,
    diagnostics: rep.diagnostics,
    indexBytes: rep.indexBytes,
    limitations: rep.limitations,
    blobKeys: Object.keys(rep.blobs),
    shapeKeys: Object.keys(rep.shapes),
    blobCoordinates: Object.keys(rep.blobs).map((k) => ({ key: k, source: rep.blobs[k].source })),
    shapeCoordinates: Object.keys(rep.shapes).map((k) => ({ key: k, source: rep.shapes[k].source })),
  }];
}

function claudeInspect() {
  const mt = require("minitrace");
  const path = REAL_FILES[0];
  const rep = mt.native().File(path).Inspect();
  const byStatus = {};
  const incomplete = [];
  for (let i = 0; i < rep.requests.length; i++) {
    const r = rep.requests[i];
    byStatus[r.status] = (byStatus[r.status] || 0) + 1;
    if (r.status !== "complete" && incomplete.length < 40) {
      incomplete.push({
        id: r.id,
        line: r.source.line,
        shapeHash: r.shapeHash,
        status: r.status,
        refs: r.refs,
        diagnostics: r.diagnostics,
      });
    }
  }
  return [{
    path: path,
    scan: rep.scan,
    requestCount: rep.requests.length,
    statusCounts: byStatus,
    incompleteSample: incomplete,
    compactionCount: rep.compactions.length,
    compactions: rep.compactions,
    diagnostics: rep.diagnostics,
    indexBytes: rep.indexBytes,
    limitations: rep.limitations,
    blobCount: Object.keys(rep.blobs).length,
    shapeCount: Object.keys(rep.shapes).length,
  }];
}

__verb__("scanAll", {
  name: "scan-all",
  short: "Stream every real native transcript and report scan evidence",
  fields: { options: { bind: "options" } },
});

__verb__("earlyStop", {
  name: "early-stop",
  short: "Demonstrate hashScope consumed_prefix after early stop",
  fields: { options: { bind: "options" } },
});

__verb__("diagnosticsDemo", {
  name: "oversized",
  short: "Surface oversized-record diagnostics with a tiny record budget",
  fields: { options: { bind: "options" } },
});

__verb__("malformedOnly", {
  name: "malformed",
  short: "Surface malformed-record diagnostics on the synthetic fixture",
  fields: { options: { bind: "options" } },
});

__verb__("synthInspect", {
  name: "synth-inspect",
  short: "Inspect synthetic native fixture: requests, blobs, compactions",
  fields: { options: { bind: "options" } },
});

__verb__("claudeInspect", {
  name: "claude-inspect",
  short: "Inspect a real Claude native source with api-request records",
  fields: { options: { bind: "options" } },
});

function inspectList(paths) {
  const mt = require("minitrace");
  const out = [];
  for (let i = 0; i < paths.length; i++) {
    const rep = mt.native().File(paths[i]).Inspect();
    const byStatus = {};
    for (let j = 0; j < rep.requests.length; j++) {
      const r = rep.requests[j];
      byStatus[r.status] = (byStatus[r.status] || 0) + 1;
    }
    out.push({
      path: paths[i],
      scan: rep.scan,
      requestCount: rep.requests.length,
      statusCounts: byStatus,
      compactionCount: rep.compactions.length,
      compactions: rep.compactions,
      diagnostics: rep.diagnostics,
      blobCount: Object.keys(rep.blobs).length,
      shapeCount: Object.keys(rep.shapes).length,
      indexBytes: rep.indexBytes,
      limitations: rep.limitations,
    });
  }
  return out;
}

function nonClaudeInspect() {
  return inspectList([
    RAW + "/codex/rollout-2026-10-13T13-24-04-01a1364a-f58a-7ce3-8460-a856dacc8a1a.jsonl",
    RAW + "/codex/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl",
    RAW + "/codex/rollout-2026-11-10T11-59-20-01a1c666-5f02-7b92-984e-99a27afd7a94.jsonl",
    RAW + "/pi/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl",
    RAW + "/pi/2026-09-17T17-19-05-437Z_01a0b061-0cdd-7557-a80a-96802d3d02c5.jsonl",
    RAW + "/pi/2026-09-29T17-08-34-793Z_01a0ee23-bd68-75c1-a81f-2c43771fc216.jsonl",
  ]);
}

function claudeOthersInspect() {
  return inspectList([
    RAW + "/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl",
    RAW + "/claude-code/agent-ae7076749749daca0.jsonl",
  ]);
}

__verb__("nonClaudeInspect", {
  name: "nonclaude-inspect",
  short: "Inspect codex and pi native sources: request and compaction counts",
  fields: { options: { bind: "options" } },
});

__verb__("claudeOthersInspect", {
  name: "claude-others-inspect",
  short: "Inspect remaining Claude native sources",
  fields: { options: { bind: "options" } },
});

function claudeRefs() {
  const mt = require("minitrace");
  const path = REAL_FILES[0];
  const rep = mt.native().File(path).Inspect();
  const missing = {};
  const missingByRequest = [];
  let refsTotal = 0;
  for (let i = 0; i < rep.requests.length; i++) {
    const r = rep.requests[i];
    refsTotal += r.refs.length;
    const m = [];
    for (let k = 0; k < r.diagnostics.length; k++) {
      const d = r.diagnostics[k];
      if (d.code === "missing_blob") {
        missing[d.reference] = (missing[d.reference] || 0) + 1;
        m.push(d.reference);
      }
    }
    missingByRequest.push({ line: r.source.line, id: r.id, refs: r.refs.length, missing: m.length, status: r.status });
  }
  const keys = Object.keys(missing);
  return [{
    path: path,
    requests: rep.requests.length,
    refsTotal: refsTotal,
    distinctMissingRefs: keys.length,
    availableBlobs: Object.keys(rep.blobs).length,
    missingRefCounts: keys.map((k) => ({ ref: k, timesReferenced: missing[k], everAvailable: Object.prototype.hasOwnProperty.call(rep.blobs, k) })),
    missingByRequest: missingByRequest,
    limitations: rep.limitations,
  }];
}

__verb__("claudeRefs", {
  name: "claude-refs",
  short: "Exact missing-blob reference diagnostics for the real Claude source",
  fields: { options: { bind: "options" } },
});

function codexCompacted() {
  const mt = require("minitrace");
  const paths = [REAL_FILES[4], REAL_FILES[5]];
  const out = [];
  for (let i = 0; i < paths.length; i++) {
    const hits = [];
    const stats = mt.native().File(paths[i]).ForEach((rec) => {
      if (!rec.diagnostic && rec.value && rec.value.type === "compacted") {
        hits.push({
          line: rec.source.line,
          byteOffset: rec.source.byteOffset,
          rawBytes: rec.source.rawBytes,
          payloadKeys: Object.keys(rec.value.payload || {}),
          payloadType: rec.value.payload && rec.value.payload.type,
          hasReplacement: !!(rec.value.payload && rec.value.payload.replacement_history),
        });
      }
      return true;
    });
    out.push({ path: paths[i], scan: stats, compactedRecords: hits });
  }
  return out;
}

__verb__("codexCompacted", {
  name: "codex-compacted",
  short: "Physical coordinates of codex compacted records (not native compaction objects)",
  fields: { options: { bind: "options" } },
});

function numericFidelity() {
  const mt = require("minitrace");
  let hit = null;
  mt.native().File(SYNTH).ForEach((rec) => {
    if (!rec.diagnostic && rec.value && rec.value.type === "last") {
      hit = {
        line: rec.source.line,
        rawJSON: rec.rawJSON,
        decodedExact: rec.value.exact,
        decodedString: String(rec.value.exact),
      };
    }
    return true;
  });
  return [{ syntheticLastRecord: hit, note: "decoded JS number is not exact; rawJSON is the evidence" }];
}

__verb__("numericFidelity", {
  name: "numeric-fidelity",
  short: "Show large-integer precision loss in decoded value vs rawJSON",
  fields: { options: { bind: "options" } },
});
