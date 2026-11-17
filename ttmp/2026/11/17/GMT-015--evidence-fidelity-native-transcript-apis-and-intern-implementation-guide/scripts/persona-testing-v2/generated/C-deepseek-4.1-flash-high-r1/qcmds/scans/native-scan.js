__section__("filters", {
  title: "Filters",
  fields: {
    includePaths: {
      type: "stringList",
      help: "Absolute paths to scan (native JSONL)",
      default: [],
    },
  },
});

function nativeScan(filters) {
  const mt = require("minitrace");
  const paths = filters.includePaths || [];
  const rows = [];
  for (const path of paths) {
    const counters = { malformed: 0, oversized: 0, invalidUtf8: 0, apiRequests: 0, total: 0 };
    const diagCodes = {};
    let forEachError = null;
    let scan = null;
    try {
      scan = mt.native().File(path).MaxRecordBytes(16 * 1024 * 1024).ForEach(rec => {
        counters.total++;
        if (rec.diagnostic) {
          const c = rec.diagnostic.code || "unknown";
          diagCodes[c] = (diagCodes[c] || 0) + 1;
          if (c.indexOf("oversize") >= 0) counters.oversized++;
          else if (c.indexOf("utf8") >= 0 || c.indexOf("utf-8") >= 0) counters.invalidUtf8++;
          else counters.malformed++;
          return;
        }
        if (rec.value && rec.value.type === "api-request") counters.apiRequests++;
      });
    } catch (e) {
      forEachError = String(e && e.message ? e.message : e);
    }

    let inspect = null, inspectError = null;
    const statusCounts = {};
    let complete = 0, incomplete = 0, requestCount = 0;
    const incompleteSamples = [];
    try {
      inspect = mt.native().File(path).MaxRecordBytes(16 * 1024 * 1024).Inspect();
      requestCount = (inspect.requests || []).length;
      for (const r of inspect.requests || []) {
        statusCounts[r.status] = (statusCounts[r.status] || 0) + 1;
        if (r.status === "complete") complete++;
        else {
          incomplete++;
          if (incompleteSamples.length < 5) {
            incompleteSamples.push({
              id: r.id,
              line: r.source && r.source.line,
              status: r.status,
              diagnostics: (r.diagnostics || []).map(d => (typeof d === "string" ? d : (d.code || JSON.stringify(d)))),
            });
          }
        }
      }
    } catch (e) {
      inspectError = String(e && e.message ? e.message : e);
    }

    rows.push({
      path: path,
      forEach: scan ? {
        sha256: scan.sha256, hashScope: scan.hashScope, bytes: scan.bytes,
        lines: scan.lines, malformed: scan.malformed, stopReason: scan.stopReason,
      } : null,
      forEachError: forEachError,
      counters: counters,
      diagCodes: diagCodes,
      inspect: inspect ? {
        sha256: inspect.scan.sha256, hashScope: inspect.scan.hashScope,
        bytes: inspect.scan.bytes, lines: inspect.scan.lines,
        malformed: inspect.scan.malformed, stopReason: inspect.scan.stopReason,
        blobs: (inspect.blobs ? Object.keys(inspect.blobs).length : null),
        shapes: (inspect.shapes ? Object.keys(inspect.shapes).length : null),
        compactions: (inspect.compactions || []).length,
        diagnostics: (inspect.diagnostics || []).length,
        indexBytes: inspect.indexBytes,
        limitations: inspect.limitations || null,
      } : null,
      inspectError: inspectError,
      requests: { count: requestCount, complete: complete, incomplete: incomplete, statusCounts: statusCounts, incompleteSamples: incompleteSamples },
    });
  }
  return rows;
}

__verb__("nativeScan", {
  name: "native-scan",
  short: "Scan native JSONL with mt.native (per file request graph counts)",
  fields: { filters: { bind: "filters" } },
});
