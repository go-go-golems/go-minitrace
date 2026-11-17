function realInspect() {
  const mt = require("minitrace");
  const paths = [
    "/private/tmp/gmt015-persona-v2-matrix/corpus/raw/claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl",
    "/private/tmp/gmt015-persona-v2-matrix/corpus/raw/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl",
    "/private/tmp/gmt015-persona-v2-matrix/corpus/raw/claude-code/agent-ae7076749749daca0.jsonl",
    "/private/tmp/gmt015-persona-v2-matrix/corpus/raw/codex/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl",
    "/private/tmp/gmt015-persona-v2-matrix/corpus/raw/codex/rollout-2026-10-13T13-24-04-01a1364a-f58a-7ce3-8460-a856dacc8a1a.jsonl",
    "/private/tmp/gmt015-persona-v2-matrix/corpus/raw/codex/rollout-2026-11-10T11-59-20-01a1c666-5f02-7b92-984e-99a27afd7a94.jsonl",
    "/private/tmp/gmt015-persona-v2-matrix/corpus/raw/pi/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl",
    "/private/tmp/gmt015-persona-v2-matrix/corpus/raw/pi/2026-09-17T17-19-05-437Z_01a0b061-0cdd-7557-a80a-96802d3d02c5.jsonl",
    "/private/tmp/gmt015-persona-v2-matrix/corpus/raw/pi/2026-09-29T17-08-34-793Z_01a0ee23-bd68-75c1-a81f-2c43771fc216.jsonl",
  ];
  const results = [];
  for (const p of paths) {
    const out = { path: p };
    try {
      const insp = mt.native().File(p)
        .MaxRecordBytes(64 * 1024 * 1024)
        .MaxIndexBytes(512 * 1024 * 1024)
        .Inspect();
      out.ok = true;
      out.scan = {
        sha256: insp.scan.sha256,
        hashScope: insp.scan.hashScope,
        bytes: insp.scan.bytes,
        lines: insp.scan.lines,
        malformed: insp.scan.malformed,
        stopReason: insp.scan.stopReason,
      };
      out.requestCount = insp.requests.length;
      out.complete = insp.requests.filter(r => r.status === "complete").length;
      out.incomplete = insp.requests.filter(r => r.status !== "complete").length;
      out.statusCounts = {};
      for (const r of insp.requests) {
        out.statusCounts[r.status] = (out.statusCounts[r.status] || 0) + 1;
      }
      out.compactions = insp.compactions.length;
      out.blobs = Object.keys(insp.blobs).length;
      out.shapes = Object.keys(insp.shapes).length;
      out.indexBytes = insp.indexBytes;
      out.diagnostics = insp.diagnostics;
      out.requestSamples = insp.requests.slice(0, 3).map(r => ({
        id: r.id, status: r.status, querySource: r.querySource ?? null, source: r.source,
      }));
    } catch (e) {
      out.ok = false;
      out.error = String(e && e.message ? e.message : e);
    }
    results.push(out);
  }
  return JSON.stringify(results, null, 2);
}

__verb__("realInspect", {
  name: "real-inspect",
  short: "Inspect real native sources request graphs and compactions",
});
