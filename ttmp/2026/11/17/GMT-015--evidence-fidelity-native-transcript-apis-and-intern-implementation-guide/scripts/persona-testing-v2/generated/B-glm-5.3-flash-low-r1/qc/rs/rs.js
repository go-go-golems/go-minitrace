// Real-session summaries: one archive per framework
function rs() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory()
    .MaxCellBytes(16000).StrictEvidence(true).Build();
  try {
    const counts = db.execute(mt.query().SessionToolCounts().Build());
    if (counts.error || !counts.complete) throw new Error(counts.error || "incomplete counts");
    const tokens = db.execute(mt.query().TokenUsageRows().Build());
    if (tokens.error || !tokens.complete) throw new Error(tokens.error || "incomplete tokens");
    const bySession = {};
    for (const t of tokens.rows) {
      (bySession[t.session_id] = bySession[t.session_id] || []).push(t);
    }
    return counts.rows.map(c => ({
      session_id: c.session_id,
      tool_count: c.tool_count,
      token_rows: (bySession[c.session_id] || []).map(r => {
        const o = {};
        for (const k of Object.keys(r)) o[k] = r[k];
        return o;
      }),
    }));
  } finally { db.close(); }
}

__verb__("rs", {
  name: "rs",
  short: "Per-session tool counts and token usage rows",
  fields: {},
});
