// Native scan command: uses mt.native() to scan real raw JSONL files and
// the synthetic native.jsonl fixture.
__verb__("scanReal", {
  name: "scan-real",
  short: "Scan the nine real native raw sources with mt.native()",
});

function scanReal() {
  return scanInner(false);
}

function scanInner(synthetic) {
  const mt = require("minitrace");
  const rawBase = "/private/tmp/gmt015-persona-v2-matrix/corpus/raw/";
  const realFiles = [
    "claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl",
    "claude-code/agent-ae7076749749daca0.jsonl",
    "claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl",
    "codex/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl",
    "codex/rollout-2026-10-13T13-24-04-01a1364a-f58a-7ce3-8460-a856dacc8a1a.jsonl",
    "codex/rollout-2026-11-10T11-59-20-01a1c666-5f02-7b92-984e-99a27afd7a94.jsonl",
    "pi/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl",
    "pi/2026-09-17T17-19-05-437Z_01a0b061-0cdd-7557-a80a-96802d3d02c5.jsonl",
    "pi/2026-09-29T17-08-34-793Z_01a0ee23-bd68-75c1-a81f-2c43771fc216.jsonl",
  ];
  const files = synthetic
    ? ["/private/tmp/gmt015-persona-v2-matrix/fixtures/native.jsonl"]
    : realFiles.map(f => rawBase + f);
  const rows = [];
  for (const f of files) {
    let apiRequests = 0;
    let compactEvents = 0;
    let diagnostics = 0;
    const typeCounts = {};
    const scan = mt.native().File(f)
      .MaxRecordBytes(16 * 1024 * 1024)
      .ForEach(record => {
        if (record.diagnostic) { diagnostics++; return; }
        const t = record.value && record.value.type;
        if (t) typeCounts[t] = (typeCounts[t] || 0) + 1;
        if (t === "api-request") apiRequests++;
        if (t === "compaction" || t === "compact" || (record.value.isCompactSummary)) compactEvents++;
      });
    const out = {
      file: f,
      sha256: scan.sha256,
      hashScope: scan.hashScope,
      bytes: scan.bytes,
      lines: scan.lines,
      malformed: scan.malformed,
      stopReason: scan.stopReason,
      diagnostics: diagnostics,
      apiRequestRecords: apiRequests,
      typeCounts: typeCounts,
    };
    if (synthetic) {
      const insp = mt.native().File(f).Inspect();
      const req = insp.requests.map(r => ({ id: r.id, status: r.status, diagnostics: r.diagnostics }));
      out.requests = req;
      out.complete = req.filter(r => r.status === "complete").length;
      out.incomplete = req.filter(r => r.status !== "complete").length;
      out.scanFromInspect = {
        sha256: insp.scan.sha256, hashScope: insp.scan.hashScope,
        bytes: insp.scan.bytes, lines: insp.scan.lines, malformed: insp.scan.malformed,
      };
      out.compactions = insp.compactions.length;
    }
    rows.push(out);
  }
  return { columns: ["result"], rows: [{ result: JSON.stringify(rows) }], count: rows.length, complete: true };
}

__verb__("scanSynthetic", {
  name: "scan-synthetic",
  short: "Scan the synthetic native.jsonl fixture",
});

function scanSynthetic() {
  return scanInner(true);
}
