function realSessionSummary() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try {
    const out = {};
    const sess = db.query("SELECT session_id, agent_framework FROM sessions");
    out.sessions = sess;
    const counts = db.execute(mt.query().SessionToolCounts().Build());
    if (counts.error) { out.count_error = counts.error; }
    out.tool_counts = counts.rows;
    const tok = db.execute(mt.query().TokenUsageRows().Build());
    if (tok.error) { out.token_error = tok.error; out.token_rows = null; }
    else {
      const rows = tok.rows || [];
      const agg = {};
      for (const r of rows) {
        const o = r.token_origin;
        if (!agg[o]) agg[o] = { rows: 0, input_tokens: 0, output_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 0, reasoning_tokens: 0, total_tokens: 0, zero_fill_rows: 0, distinct_methods: [] };
        agg[o].rows++;
        agg[o].input_tokens += r.input_tokens;
        agg[o].output_tokens += r.output_tokens;
        agg[o].cache_creation_tokens += r.cache_creation_tokens;
        agg[o].cache_read_tokens += r.cache_read_tokens;
        agg[o].reasoning_tokens += r.reasoning_tokens;
        agg[o].total_tokens += r.total_tokens;
        if (r.reported_usage_fields === 0) agg[o].zero_fill_rows++;
        if (agg[o].distinct_methods.indexOf(r.token_method) < 0) agg[o].distinct_methods.push(r.token_method);
      }
      out.token_evidence = { scope: "turn", token_unit: rows.length ? rows[0].token_unit : null, by_origin: agg, row_count: rows.length, complete: tok.complete === true };
    }
    return out;
  } finally { db.close(); }
}

__verb__("realSessionSummary", {
  name: "real-session-summary",
  short: "Session, tool count and token evidence summary for loaded archives",
  fields: {}
});
