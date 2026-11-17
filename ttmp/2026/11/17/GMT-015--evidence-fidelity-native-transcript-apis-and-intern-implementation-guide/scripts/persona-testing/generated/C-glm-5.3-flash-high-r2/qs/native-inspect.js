__section__("filters", {
  title: "Modes",
  fields: {
    mode: { type: "string", default: "all", help: "all | earlystop | oversized" },
  },
});

function _reqSummary(r) {
  return {
    id: r.id,
    status: r.status,
    shapeHash: r.shapeHash,
    keep: r.keep === undefined ? null : r.keep,
    refCount: (r.refs || []).length,
    diagnostics: (r.diagnostics || []).map((d) => d.code + ": " + String(d.message).slice(0, 120)),
    source: r.source,
  };
}

function _compactSummary(c) {
  return {
    boundaryUuid: c.boundaryUuid,
    parentUuid: c.parentUuid,
    logicalParentUuid: c.logicalParentUuid,
    source: c.source,
    preceding: c.precedingMessage ? { type: c.precedingMessage.type, source: c.precedingMessage.source } : null,
    following: c.followingMessage ? { type: c.followingMessage.type, source: c.followingMessage.source } : null,
    followingAssistant: c.followingAssistant ? { type: c.followingAssistant.type, source: c.followingAssistant.source } : null,
    summaryKeys: c.summary ? Object.keys(c.summary) : null,
    compactMetadataKeys: c.compactMetadata ? Object.keys(c.compactMetadata) : null,
  };
}

function inspect(filters) {
  const mt = require("minitrace");
  const mode = filters.mode || "all";
  const out = {};

  if (mode === "all" || mode === "fixture") {
    const ev = mt.native().File("/private/tmp/gmt015-persona/inputs/native.jsonl").Inspect();
    out.fixture = {
      scan: { sha256: ev.scan.sha256, hashScope: ev.scan.hashScope, bytes: ev.scan.bytes, lines: ev.scan.lines, malformed: ev.scan.malformed, stopReason: ev.scan.stopReason },
      requests: ev.requests.map(_reqSummary),
      compactions: ev.compactions.map(_compactSummary),
      diagnostics: ev.diagnostics,
      indexBytes: ev.indexBytes,
      limitations: ev.limitations,
    };
  }

  if (mode === "all" || mode === "claude") {
    const ev = mt.native().File("/private/tmp/gmt015-persona/corpus/raw/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl").Inspect();
    out.claude_d047a697 = {
      scan: { sha256: ev.scan.sha256, hashScope: ev.scan.hashScope, bytes: ev.scan.bytes, lines: ev.scan.lines, malformed: ev.scan.malformed },
      requestCount: ev.requests.length,
      complete: ev.requests.filter((r) => r.status === "complete").map(_reqSummary),
      incomplete: ev.requests.filter((r) => r.status !== "complete").map(_reqSummary),
      compactions: ev.compactions.map(_compactSummary),
      diagnostics: ev.diagnostics,
      indexBytes: ev.indexBytes,
      limitations: ev.limitations,
    };
  }

  if (mode === "all" || mode === "earlystop") {
    let seen = 0;
    const s = mt.native().File("/private/tmp/gmt015-persona/corpus/raw/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl").ForEach(() => {
      seen++;
      if (seen >= 3) return false;
      return;
    });
    out.earlyStop = { recordsSeen: seen, bytes: s.bytes, lines: s.lines, sha256: s.sha256, hashScope: s.hashScope, stopReason: s.stopReason };
  }

  if (mode === "all" || mode === "oversized") {
    const s = mt.native().File("/private/tmp/gmt015-persona/corpus/raw/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl").MaxRecordBytes(64).ForEach((r) => {
      if (r.diagnostic && r.diagnostic.code === "oversized_record") {
        out.oversizedSample = { line: r.source.line, byteOffset: r.source.byteOffset, code: r.diagnostic.code, message: String(r.diagnostic.message).slice(0, 160) };
        return false;
      }
      return;
    });
    out.oversizedScan = { bytes: s.bytes, lines: s.lines, hashScope: s.hashScope, stopReason: s.stopReason };
  }

  return out;
}

__verb__("inspect", {
  name: "inspect",
  short: "mt.native().Inspect() on fixture + real Claude source; early-stop and oversized demos",
  fields: { filters: { bind: "filters" } },
});
