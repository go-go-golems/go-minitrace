// native-bounds.js — demonstrate early stop (prefix hash scope) and
// malformed/oversized record diagnostics without evaluating record content.
__section__("filters", {
  title: "Native bounds",
  fields: {
    file: { type: "string", help: "Absolute native JSONL path", required: true },
    stopAfter: { type: "int", default: 0, help: "Stop after N records (0 = EOF)" },
    maxRecordBytes: { type: "int", default: 16777216, help: "Per-record byte cap" },
  },
});

function boundsDemo(filters) {
  const mt = require("minitrace");
  let seen = 0;
  const diags = [];
  const observed = [];
  const scan = mt.native().File(filters.file)
    .MaxRecordBytes(filters.maxRecordBytes || 16777216)
    .ForEach(rec => {
      if (rec.diagnostic) {
        if (diags.length < 10) diags.push({ line: rec.source.line, byteOffset: rec.source.byteOffset, rawBytes: rec.source.rawBytes, code: rec.diagnostic.code, message: rec.diagnostic.message });
      } else {
        seen++;
        if (observed.length < 10) observed.push({ line: rec.source.line, byteOffset: rec.source.byteOffset, rawBytes: rec.source.rawBytes, type: rec.value.type, hasRawJSON: typeof rec.rawJSON === "string" });
      }
      if (filters.stopAfter && (seen + diags.length) >= filters.stopAfter) return false;
      return true;
    });
  return {
    path: filters.file,
    stopAfter: filters.stopAfter || 0,
    maxRecordBytes: filters.maxRecordBytes || 16777216,
    sha256: scan.sha256, hashScope: scan.hashScope, bytes: scan.bytes,
    lines: scan.lines, malformed: scan.malformed, stopReason: scan.stopReason,
    parsedRecordsSeen: seen, diagnostics: diags, firstRecords: observed,
  };
}

__verb__("boundsDemo", {
  name: "bounds-demo",
  short: "Early stop (consumed_prefix) and malformed/oversized record diagnostics",
  fields: { filters: { bind: "filters" } },
});
