function nativeScan() {
  const mt = require("minitrace");
  const paths = [
    "/private/tmp/gmt015-persona-v2-matrix/fixtures/native.jsonl",
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
    let apiRequests = 0;
    let diagnostics = 0;
    const typeCounts = {};
    let diagnosticCodes = {};
    const scan = mt.native().File(p)
      .MaxRecordBytes(64 * 1024 * 1024)
      .ForEach(record => {
        if (record.diagnostic) {
          diagnostics++;
          diagnosticCodes[record.diagnostic.code] = (diagnosticCodes[record.diagnostic.code] || 0) + 1;
          return;
        }
        const t = record.value && record.value.type;
        if (t !== undefined) {
          typeCounts[String(t)] = (typeCounts[String(t)] || 0) + 1;
        }
        if (t === "api-request") {
          apiRequests++;
        }
      });
    const row = {
      path: p,
      sha256: scan.sha256,
      hashScope: scan.hashScope,
      bytes: scan.bytes,
      lines: scan.lines,
      malformed: scan.malformed,
      stopReason: scan.stopReason,
      apiRequests: apiRequests,
      diagnostics: diagnostics,
      diagnosticCodes: diagnosticCodes,
      typeCounts: typeCounts,
    };
    results.push(row);
  }
  return JSON.stringify(results, null, 2);
}

__verb__("nativeScan", {
  name: "native-scan",
  short: "Scan native JSONL sources with mt.native()",
});
