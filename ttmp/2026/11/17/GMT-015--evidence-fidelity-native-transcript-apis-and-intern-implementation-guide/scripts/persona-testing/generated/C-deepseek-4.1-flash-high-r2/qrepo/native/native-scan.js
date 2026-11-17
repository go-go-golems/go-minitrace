// native-scan.js — streaming mt.native() scan of explicit native JSONL files.
// Records are counted structurally; no transcript content is executed.
__section__("filters", {
  title: "Native scan options",
  fields: {
    files: { type: "stringList", help: "Absolute native JSONL paths" },
    maxRecordBytes: { type: "int", default: 16777216, help: "Per-record byte cap" },
  },
});

const countTypes = function(rec, acc) {
  if (rec.diagnostic) { acc.bad.push({line: rec.source.line, code: rec.diagnostic.code, message: rec.diagnostic.message, byteOffset: rec.source.byteOffset, rawBytes: rec.source.rawBytes}); return; }
  const t = (rec.value && rec.value.type) || "(untyped)";
  acc.types[t] = (acc.types[t] || 0) + 1;
  acc.count++;
};

function scanFiles(filters) {
  const mt = require("minitrace");
  const results = [];
  for (const p of filters.files || []) {
    const acc = { count: 0, types: {}, bad: [] };
    const scan = mt.native().File(p).MaxRecordBytes(filters.maxRecordBytes || 16777216)
      .ForEach(rec => { countTypes(rec, acc); });
    results.push({
      path: p, sha256: scan.sha256, hashScope: scan.hashScope,
      bytes: scan.bytes, lines: scan.lines,
      malformed: scan.malformed, stopReason: scan.stopReason,
      records: acc.count, types: acc.types, diagnostics: acc.bad,
    });
  }
  return results;
}

__verb__("scanFiles", {
  name: "scan-files",
  short: "Stream native JSONL, count records, report whole-source vs prefix hash scope",
  fields: { filters: { bind: "filters" } },
});
