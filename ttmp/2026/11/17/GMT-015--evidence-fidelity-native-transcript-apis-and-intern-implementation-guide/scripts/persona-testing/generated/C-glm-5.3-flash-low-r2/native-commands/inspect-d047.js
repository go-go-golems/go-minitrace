__verb__("inspectD047", { name: "inspect-d047", short: "Inspect() d047a697 claude file" });
function inspectD047() {
  const mt = require("minitrace");
  const ev = mt.native().File("/private/tmp/gmt015-persona/corpus/raw/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl").Inspect();
  const counts = {};
  ev.requests.forEach(r => counts[r.status] = (counts[r.status] || 0) + 1);
  return { scan: ev.scan, requestCount: ev.requests.length, statusCounts: counts, compactions: ev.compactions.length, diagnostics: ev.diagnostics,
    incomplete: ev.requests.filter(r => r.status !== "complete").slice(0, 3).map(r => ({ id: r.id, status: r.status, diagnostics: r.diagnostics, source: r.source })) };
}
