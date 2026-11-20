function fieldprobe() {
  const mt = require("minitrace");
  const out = [];
  const picks = [
    ["claude-code", "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/claude-code/active/2026-10/7dce992c-7793-4e1a-950c-4db92cef2502.minitrace.json"],
    ["codex", "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/codex/active/2026-10/01a13157-f631-7cd3-8f84-e2666b24af01.minitrace.json"],
    ["pi", "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/pi/active/2026-09/01a0a5c3-342f-7447-be74-b6c35f542ff7.minitrace.json"],
  ];
  for (const [fw, path] of picks) {
    const db = mt.db().File(path).SQLiteMemory().Build();
    try {
      const tok = db.execute(mt.query().TokenUsageRows().Build());
      if (tok.error || !tok.complete) { out.push({ framework: fw, error: String(tok.error || "incomplete") }); continue; }
      const distinct = {};
      for (const r of tok.rows) {
        const k = r.reported_usage_fields === null ? "null" : JSON.stringify(r.reported_usage_fields);
        distinct[k] = (distinct[k] || 0) + 1;
      }
      out.push({ framework: fw, distinct_reported_usage_fields: JSON.stringify(distinct) });
    } finally { db.close(); }
  }
  return out;
}

__verb__("fieldprobe", {
  name: "fieldprobe",
  short: "Distinct reported_usage_fields per archive",
  fields: {},
});
