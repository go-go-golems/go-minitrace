function edgeProbe() {
  const mt = require("minitrace");
  const p = "/private/tmp/gmt015-persona-v2-matrix/fixtures/native.jsonl";
  const out = {};
  // 1. early stop at first record
  const scan1 = mt.native().File(p).MaxRecordBytes(64 * 1024 * 1024).ForEach(r => false);
  out.earlyStop = { sha256: scan1.sha256, hashScope: scan1.hashScope, bytes: scan1.bytes, lines: scan1.lines, stopReason: scan1.stopReason };
  // 2. tiny record limit -> oversized diagnostics
  let oversized = 0, diagCodes = {};
  const scan2 = mt.native().File(p).MaxRecordBytes(32).ForEach(r => {
    if (r.diagnostic) { oversized++; diagCodes[r.diagnostic.code] = (diagCodes[r.diagnostic.code] || 0) + 1; }
  });
  out.oversized = { scan: { sha256: scan2.sha256, hashScope: scan2.hashScope, bytes: scan2.bytes, lines: scan2.lines, malformed: scan2.malformed, stopReason: scan2.stopReason }, oversizedCount: oversized, diagCodes };
  return JSON.stringify(out, null, 2);
}

__verb__("edgeProbe", {
  name: "edge-probe",
  short: "Probe early-stop hash scope and oversized-record diagnostics",
});
