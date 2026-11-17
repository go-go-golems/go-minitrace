__section__("filters", {
  title: "Filters",
  fields: {
    only: { type: "string", help: "optional basename filter" }
  }
});

function nativeScan(filters) {
  const mt = require("minitrace");
  const base = "/private/tmp/gmt015-persona-v2-matrix/corpus/raw";
  const real = [
    "claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl",
    "claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl",
    "claude-code/agent-ae7076749749daca0.jsonl",
    "codex/rollout-2026-10-13T13-24-04-01a1364a-f58a-7ce3-8460-a856dacc8a1a.jsonl",
    "codex/rollout-2026-11-10T11-59-20-01a1c666-5f02-7b92-984e-99a27afd7a94.jsonl",
    "codex/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl",
    "pi/2026-09-17T17-19-05-437Z_01a0b061-0cdd-7557-a80a-96802d3d02c5.jsonl",
    "pi/2026-09-29T17-08-34-793Z_01a0ee23-bd68-75c1-a81f-2c43771fc216.jsonl",
    "pi/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl"
  ];
  const synth = "/private/tmp/gmt015-persona-v2-matrix/fixtures/native.jsonl";
  const jobs = real.map(function(rel) { return { kind: "real", rel: rel, path: base + "/" + rel }; });
  jobs.push({ kind: "synthetic", rel: "native.jsonl", path: synth });

  const out = [];
  jobs.forEach(function(job) {
    if (filters && filters.only && job.rel.indexOf(filters.only) < 0) return;
    const typeCounts = {};
    let records = 0;
    let malformed = 0;
    const scan = mt.native().File(job.path)
      .MaxRecordBytes(16 * 1024 * 1024)
      .ForEach(function(r) {
        records++;
        if (r.diagnostic) { malformed++; return; }
        const t = r.value && r.value.type ? String(r.value.type) : "<none>";
        typeCounts[t] = (typeCounts[t] || 0) + 1;
      });
    const insp = mt.native().File(job.path)
      .MaxRecordBytes(16 * 1024 * 1024)
      .MaxIndexBytes(256 * 1024 * 1024)
      .Inspect();
    let complete = 0, incomplete = 0;
    const statusCounts = {};
    const incompletes = [];
    (insp.requests || []).forEach(function(req) {
      const st = req.status || "<none>";
      statusCounts[st] = (statusCounts[st] || 0) + 1;
      if (st === "complete") complete++; else {
        incomplete++;
        if (incompletes.length < 5) incompletes.push({ id: req.id, status: st, diagnostics: req.diagnostics });
      }
    });
    out.push({
      kind: job.kind,
      rel: job.rel,
      path: job.path,
      scan: { sha256: scan.sha256, hashScope: scan.hashScope, bytes: scan.bytes, lines: scan.lines, malformed: scan.malformed, stopReason: scan.stopReason },
      records: records,
      malformedRecords: malformed,
      typeCounts: typeCounts,
      requestsTotal: (insp.requests || []).length,
      requestsComplete: complete,
      requestsIncomplete: incomplete,
      requestStatusCounts: statusCounts,
      incompleteSamples: incompletes,
      compactions: (insp.compactions || []).length,
      diagnostics: (insp.diagnostics || []).length,
      indexBytes: insp.indexBytes,
      limitations: insp.limitations
    });
  });
  return out;
}

__verb__("nativeScan", {
  name: "native-scan",
  short: "Scan real and synthetic native JSONL with mt.native",
  fields: { filters: { bind: "filters" } }
});
