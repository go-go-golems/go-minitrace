// native-inspect.js — bounded native index via mt.native().Inspect().
// Emits request graph status, exact missing-reference diagnostics, compaction
// coordinates. It never evaluates transcript content.
__section__("filters", {
  title: "Native inspect",
  fields: {
    files: { type: "stringList", help: "Absolute native JSONL paths" },
    maxRecordBytes: { type: "int", default: 16777216 },
    maxIndexBytes: { type: "int", default: 67108864 },
    showRequests: { type: "bool", default: false, help: "Include per-request status rows" },
  },
});

function inspectFiles(filters) {
  const mt = require("minitrace");
  const out = [];
  for (const p of filters.files || []) {
    let ev;
    try {
      ev = mt.native().File(p)
        .MaxRecordBytes(filters.maxRecordBytes || 16777216)
        .MaxIndexBytes(filters.maxIndexBytes || 67108864)
        .Inspect();
    } catch (e) {
      out.push({ path: p, error: String(e && e.message ? e.message : e) });
      continue;
    }
    const st = {};
    const reqs = [];
    for (const r of ev.requests || []) {
      st[r.status] = (st[r.status] || 0) + 1;
      reqs.push({
        id: r.id, querySource: r.querySource, status: r.status,
        line: r.source && r.source.line, byteOffset: r.source && r.source.byteOffset,
        shapeHash: r.shapeHash, refs: r.refs,
        diagnostics: (r.diagnostics || []).map(d => ({ code: d.code, message: d.message, reference: d.reference || null })),
      });
    }
    out.push({
      path: p,
      scan: {
        sha256: ev.scan.sha256, hashScope: ev.scan.hashScope, bytes: ev.scan.bytes,
        lines: ev.scan.lines, malformed: ev.scan.malformed, stopReason: ev.scan.stopReason,
      },
      requestCount: (ev.requests || []).length,
      statusCounts: st,
      blobCount: (ev.blobs || []).length,
      shapeCount: (ev.shapes || []).length,
      compactions: (ev.compactions || []).map(c => ({ ...c })),
      diagnostics: (ev.diagnostics || []).map(d => ({ ...d })),
      indexBytes: ev.indexBytes,
      limitations: ev.limitations,
      requests: filters.showRequests ? reqs : undefined,
    });
  }
  return out;
}

__verb__("inspectFiles", {
  name: "inspect-files",
  short: "Build bounded native index: request status, missing refs, compaction coordinates",
  fields: { filters: { bind: "filters" } },
});
