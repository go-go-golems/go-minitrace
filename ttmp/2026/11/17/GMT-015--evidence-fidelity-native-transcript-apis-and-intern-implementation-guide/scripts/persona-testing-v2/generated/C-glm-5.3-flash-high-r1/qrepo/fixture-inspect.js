function fixtureInspect() {
  const mt = require("minitrace");
  const insp = mt.native().File("/private/tmp/gmt015-persona-v2-matrix/fixtures/native.jsonl")
    .MaxRecordBytes(64 * 1024 * 1024)
    .MaxIndexBytes(64 * 1024 * 1024)
    .Inspect();
  const reqs = insp.requests.map(r => ({
    id: r.id,
    status: r.status,
    querySource: r.querySource ?? null,
    source: r.source,
    diagnostics: r.diagnostics,
  }));
  return JSON.stringify({
    scan: {
      sha256: insp.scan.sha256,
      hashScope: insp.scan.hashScope,
      bytes: insp.scan.bytes,
      lines: insp.scan.lines,
      malformed: insp.scan.malformed,
      stopReason: insp.scan.stopReason,
    },
    requestCount: insp.requests.length,
    complete: insp.requests.filter(r => r.status === "complete").length,
    incomplete: insp.requests.filter(r => r.status !== "complete").length,
    requests: reqs,
    diagnostics: insp.diagnostics,
    indexBytes: insp.indexBytes,
    limitations: insp.limitations,
  }, null, 2);
}

__verb__("fixtureInspect", {
  name: "fixture-inspect",
  short: "Inspect fixture native.jsonl request graphs",
});
