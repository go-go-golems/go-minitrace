__section__("filters", {
  title: "Modes",
  fields: {
    mode: { type: "string", default: "all", help: "all | codes | rawcompact" },
  },
});

function diag(filters) {
  const mt = require("minitrace");
  const mode = filters.mode || "all";
  const out = {};

  if (mode === "all" || mode === "codes") {
    const codes = {};
    let first = null;
    const s = mt.native().File("/private/tmp/gmt015-persona/corpus/raw/claude-code/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl").MaxRecordBytes(64).ForEach((r) => {
      if (r.diagnostic) {
        codes[r.diagnostic.code] = (codes[r.diagnostic.code] || 0) + 1;
        if (!first) first = { code: r.diagnostic.code, message: String(r.diagnostic.message).slice(0, 160), source: r.source };
      }
      return;
    });
    out.oversizedProbe = { maxRecordBytes: 64, codes, first, scan: { bytes: s.bytes, lines: s.lines, hashScope: s.hashScope, stopReason: s.stopReason, malformed: s.malformed } };
  }

  if (mode === "all" || mode === "rawcompact") {
    const ev = mt.native().File("/private/tmp/gmt015-persona/inputs/native.jsonl").Inspect();
    out.fixtureCompactionRaw = JSON.stringify(ev.compactions).slice(0, 2000);
    out.fixtureRequestRawSample = JSON.stringify(ev.requests[2]).slice(0, 800);
    out.fixtureKeys = ev.compactions.length ? Object.keys(ev.compactions[0]) : [];
  }

  return out;
}

__verb__("diag", {
  name: "diag",
  short: "Diagnostic probes: oversized-record codes and raw compaction object",
  fields: { filters: { bind: "filters" } },
});
