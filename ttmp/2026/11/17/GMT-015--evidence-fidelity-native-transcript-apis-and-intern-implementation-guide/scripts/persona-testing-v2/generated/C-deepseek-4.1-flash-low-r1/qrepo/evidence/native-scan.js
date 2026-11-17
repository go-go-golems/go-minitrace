__section__("inputs", {
  title: "Inputs",
  fields: {
    paths: {
      type: "stringList",
      help: "Absolute paths to native JSONL files"
    }
  }
});

function scanAll(inputs) {
  const mt = require("minitrace");
  const out = [];
  for (const p of inputs.paths) {
    let api = 0;
    let diag = 0;
    let others = {};
    const scan = mt.native().File(p).ForEach(record => {
      if (record.diagnostic) { diag++; return; }
      const t = record.value && record.value.type;
      if (t === "api-request") api++;
      else others[String(t)] = (others[String(t)] || 0) + 1;
    });
    const insp = mt.native().File(p).Inspect();
    let complete = 0;
    let incomplete = 0;
    for (const r of insp.requests) {
      if (r.status === "complete") complete++; else incomplete++;
    }
    out.push({
      path: p,
      sha256: scan.sha256,
      hashScope: scan.hashScope,
      bytes: scan.bytes,
      lines: scan.lines,
      malformed: scan.malformed,
      stopReason: scan.stopReason,
      api_request_records: api,
      forEach_diagnostics: diag,
      requests_total: insp.requests.length,
      requests_complete: complete,
      requests_incomplete: incomplete,
      compactions: insp.compactions.length,
      indexBytes: insp.indexBytes,
      record_types: others
    });
  }
  return out;
}

__verb__("scanAll", {
  name: "scan-all",
  short: "Scan native JSONL with mt.native and report request evidence",
  fields: { inputs: { bind: "inputs" } }
});
