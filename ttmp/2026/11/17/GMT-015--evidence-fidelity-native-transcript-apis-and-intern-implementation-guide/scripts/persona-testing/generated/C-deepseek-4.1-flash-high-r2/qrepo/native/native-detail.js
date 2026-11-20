// native-detail.js — aggregate incomplete request diagnostics by code + exact reference.
// Bounded: returns counts and a capped sample of missing reference ids per file.
__section__("filters", {
  title: "Native detail",
  fields: {
    files: { type: "stringList", help: "Absolute native JSONL paths" },
    sample: { type: "int", default: 5, help: "Missing-reference samples per code" },
    maxRecordBytes: { type: "int", default: 16777216 },
    maxIndexBytes: { type: "int", default: 67108864 },
  },
});

function detailFiles(filters) {
  const mt = require("minitrace");
  const out = [];
  const sampleN = Number(filters.sample || 5);
  for (const p of filters.files || []) {
    const ev = mt.native().File(p)
      .MaxRecordBytes(filters.maxRecordBytes || 16777216)
      .MaxIndexBytes(filters.maxIndexBytes || 67108864)
      .Inspect();
    const byCode = {};
    const completeIds = [];
    const incomplete = [];
    for (const r of ev.requests || []) {
      if (r.status === "complete") { completeIds.push(r.id); continue; }
      const row = { id: r.id, line: r.source.line, byteOffset: r.source.byteOffset, status: r.status, refCount: (r.refs || []).length };
      incomplete.push(row);
      for (const d of r.diagnostics || []) {
        const k = d.code + " | " + d.message;
        const b = byCode[k] || (byCode[k] = { code: d.code, message: d.message, requests: 0, references: [], truncatedRefs: 0 });
        b.requests++;
        if (d.reference) {
          if (b.references.length < sampleN) b.references.push(d.reference);
          else b.truncatedRefs++;
        }
      }
    }
    out.push({
      path: p,
      scan: { sha256: ev.scan.sha256, hashScope: ev.scan.hashScope, bytes: ev.scan.bytes, lines: ev.scan.lines, stopReason: ev.scan.stopReason, malformed: ev.scan.malformed },
      requestCount: (ev.requests || []).length,
      completeCount: completeIds.length,
      completeSample: completeIds.slice(0, sampleN),
      incompleteCount: incomplete.length,
      incompleteSample: incomplete.slice(0, sampleN),
      diagnosticGroups: Object.values(byCode),
      blobs: ev.blobs ? Object.keys(ev.blobs).length : 0,
      shapes: ev.shapes ? Object.keys(ev.shapes).length : 0,
      compactions: (ev.compactions || []).length,
      indexBytes: ev.indexBytes,
    });
  }
  return out;
}

__verb__("detailFiles", {
  name: "detail-files",
  short: "Aggregate complete/incomplete native request graphs with exact missing references",
  fields: { filters: { bind: "filters" } },
});
