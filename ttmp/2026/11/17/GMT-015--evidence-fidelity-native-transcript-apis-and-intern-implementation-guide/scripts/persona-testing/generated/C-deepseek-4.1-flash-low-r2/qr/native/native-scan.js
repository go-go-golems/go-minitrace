__section__("inputs", {
  title: "Inputs",
  fields: {
    paths: { type: "stringList", help: "Absolute native JSONL paths" },
    mode: { type: "string", default: "scan", help: "scan|inspect|early|malformed|fixture" },
    max_record_bytes: { type: "int", default: 16777216 },
  },
});

function _scanOne(path, opts) {
  const mt = require("minitrace");
  let count = 0;
  let firstTypeCounts = {};
  let diagnostics = [];
  const stats = mt.native().File(path)
    .MaxRecordBytes(opts.max_record_bytes)
    .ForEach((r) => {
      count += 1;
      if (r.diagnostic) {
        if (diagnostics.length < 20) diagnostics.push({ line: r.source.line, code: r.diagnostic.code, message: String(r.diagnostic.message).slice(0, 200) });
        return;
      }
      const t = (r.value && r.value.type) || "(none)";
      firstTypeCounts[t] = (firstTypeCounts[t] || 0) + 1;
      if (opts.stopAfter && count >= opts.stopAfter) return false;
    });
  return {
    path: path,
    sha256: stats.sha256,
    hashScope: stats.hashScope,
    bytes: stats.bytes,
    lines: stats.lines,
    malformed: stats.malformed,
    stopReason: stats.stopReason,
    recordsSeen: count,
    typeCounts: firstTypeCounts,
    diagnostics: diagnostics,
  };
}

function scan(inputs) {
  return inputs.paths.map((p) => _scanOne(p, { max_record_bytes: inputs.max_record_bytes }));
}

function early(inputs) {
  return inputs.paths.map((p) => _scanOne(p, { max_record_bytes: inputs.max_record_bytes, stopAfter: 5 }));
}

function inspect(inputs) {
  const mt = require("minitrace");
  return inputs.paths.map((p) => {
    const ev = mt.native().File(p).MaxRecordBytes(inputs.max_record_bytes).Inspect();
    const reqs = ev.requests.map((r) => ({
      id: r.id,
      source: r.source,
      shapeHash: r.shapeHash,
      refs: r.refs,
      status: r.status,
      diagnostics: r.diagnostics,
    }));
    return {
      path: p,
      scan: ev.scan,
      requestCount: ev.requests.length,
      completeRequests: ev.requests.filter((r) => r.status === "complete").length,
      incompleteRequests: ev.requests.filter((r) => r.status !== "complete").length,
      requests: reqs,
      blobCount: ev.blobs.length,
      shapeCount: ev.shapes.length,
      compactions: ev.compactions,
      diagnostics: ev.diagnostics,
      indexBytes: ev.indexBytes,
      limitations: ev.limitations,
    };
  });
}

function malformed(inputs) {
  const mt = require("minitrace");
  return inputs.paths.map((p) => {
    // Force tiny record cap so oversized records are diagnosed
    return _scanOne(p, { max_record_bytes: 64 });
  });
}

__verb__("scan", { name: "scan", short: "Stream-scan native files", fields: { inputs: { bind: "inputs" } } });
__verb__("inspect", { name: "inspect", short: "Native index inspect", fields: { inputs: { bind: "inputs" } } });
__verb__("early", { name: "early", short: "Early stop comparison", fields: { inputs: { bind: "inputs" } } });
__verb__("malformed", { name: "malformed", short: "Tiny max record diagnostic", fields: { inputs: { bind: "inputs" } } });
