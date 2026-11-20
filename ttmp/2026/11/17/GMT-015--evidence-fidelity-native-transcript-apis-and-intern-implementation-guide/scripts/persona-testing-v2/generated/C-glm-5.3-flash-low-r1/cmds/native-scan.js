__section__("paths", {
  title: "Paths",
  fields: {},
});

function nativeScan() {
  const mt = require("minitrace");
  const fs = require("fs");
  const base = "/private/tmp/gmt015-persona-v2-matrix/corpus/raw";
  const paths = [
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
  const results = [];
  for (const rel of paths) {
    const p = base + "/" + rel;
    let apiCount = 0;
    const scan = mt.native().File(p).MaxRecordBytes(64 * 1024 * 1024).ForEach(r => {
      if (r.diagnostic) return;
      if (r.value && r.value.type === "api-request") apiCount++;
    });
    const insp = mt.native().File(p).MaxRecordBytes(64 * 1024 * 1024).MaxIndexBytes(1024 * 1024 * 1024).Inspect();
    const reqStatuses = {};
    for (const rq of insp.requests) {
      reqStatuses[rq.status] = (reqStatuses[rq.status] || 0) + 1;
    }
    results.push({
      rel: rel,
      apiRequestRecords: apiCount,
      sha256: scan.sha256,
      hashScope: scan.hashScope,
      bytes: scan.bytes,
      lines: scan.lines,
      malformed: scan.malformed,
      stopReason: scan.stopReason,
      inspectScan: insp.scan,
      requestStatuses: reqStatuses,
      requestCount: insp.requests.length,
      diagnosticsCount: insp.diagnostics.length,
      limitations: insp.limitations,
    });
  }
  // synthetic fixture
  const fixPath = "/private/tmp/gmt015-persona-v2-matrix/fixtures/native.jsonl";
  const fsx = mt.native().File(fixPath).MaxRecordBytes(64 * 1024 * 1024).MaxIndexBytes(1024 * 1024 * 1024).Inspect();
  const st = {};
  for (const rq of fsx.requests) st[rq.status] = (st[rq.status] || 0) + 1;
  const scanF = fsx.scan;
  const out = { real: results, synthetic: { scan: scanF, requestStatuses: st, requestCount: fsx.requests.length, diagnosticsCount: fsx.diagnostics.length, limitations: fsx.limitations } };
  return JSON.stringify(out, null, 2);
}

__verb__("nativeScan", {
  name: "native-scan",
  short: "Scan native JSONL files",
  fields: {}
});
