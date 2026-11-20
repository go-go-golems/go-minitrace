__section__("filters", {
  title: "Modes",
  fields: {},
});

function counts(filters) {
  const mt = require("minitrace");
  const files = [
    "/private/tmp/gmt015-persona/corpus/raw/claude-code/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/claude-code/agent-ae7076749749daca0.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/codex/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/codex/rollout-2026-10-13T13-24-04-01a1364a-f58a-7ce3-8460-a856dacc8a1a.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/codex/rollout-2026-11-10T11-59-20-01a1c666-5f02-7b92-984e-99a27afd7a94.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/pi/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/pi/2026-09-17T17-19-05-437Z_01a0b061-0cdd-7557-a80a-96802d3d02c5.jsonl",
    "/private/tmp/gmt015-persona/corpus/raw/pi/2026-09-29T17-08-34-793Z_01a0ee23-bd68-75c1-a81f-2c43771fc216.jsonl",
  ];
  const out = [];
  for (const p of files) {
    const ev = mt.native().File(p).Inspect();
    const status = {};
    for (const r of ev.requests) status[r.status] = (status[r.status] || 0) + 1;
    out.push({
      file: p.split("/").pop(),
      requests: ev.requests.length,
      statusCounts: status,
      blobs: Object.keys(ev.blobs || {}).length,
      shapes: Object.keys(ev.shapes || {}).length,
      compactions: ev.compactions.length,
      diagnostics: (ev.diagnostics || []).length,
      incompleteSample: ev.requests.filter((r) => r.status !== "complete").slice(0, 2).map((r) => ({ id: r.id, status: r.status, diagnostics: (r.diagnostics || []).map((d) => d.code) })),
    });
  }
  return out;
}

__verb__("counts", {
  name: "counts",
  short: "Inspect() request/compaction counts for remaining native sources",
  fields: { filters: { bind: "filters" } },
});
