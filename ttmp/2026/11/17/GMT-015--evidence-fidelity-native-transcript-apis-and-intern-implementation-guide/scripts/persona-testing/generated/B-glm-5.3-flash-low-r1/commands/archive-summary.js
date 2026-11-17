// Per-framework tool/token summary over runtime archives.
const mt = require("minitrace");

function perArchiveSummary(filters) {
  const mt2 = require("minitrace");
  const db = mt2.db().RuntimeArchives().SQLiteMemory().MaxCellBytes(16000).Build();
  try {
    const sessions = db.query("SELECT session_id, agent_framework, turn_count, tool_call_count, started_at FROM sessions ORDER BY agent_framework, session_id");
    const out = [];
    for (const s of sessions) {
      const tokRecipe = mt2.query().TokenUsageRows().SessionID(s.session_id).Build();
      const tokRows = db.execute(tokRecipe).rows || [];
      const byOrigin = {};
      for (const t of tokRows) {
        const k = t.scope + "/" + t.token_origin + "/" + t.token_unit + "/" + t.token_method;
        if (!byOrigin[k]) byOrigin[k] = { rows: 0, total_tokens: 0 };
        byOrigin[k].rows++;
        if (t.total_tokens !== null && t.total_tokens !== undefined) byOrigin[k].total_tokens += t.total_tokens;
      }
      const tools = db.queryOne("SELECT COUNT(*) AS tools, SUM(source_ordinal IS NOT NULL) AS with_ordinal FROM tool_calls WHERE session_id = ?", [s.session_id]);
      out.push({
        session_id: s.session_id,
        framework: s.agent_framework,
        turn_count: s.turn_count,
        tool_call_count: s.tool_call_count,
        tools_with_ordinal: tools ? tools.with_ordinal : null,
        token_groups: byOrigin,
      });
    }
    return out;
  } finally {
    db.close();
  }
}

__verb__("perArchiveSummary", {
  name: "archive-summary",
  short: "Token/tool summary per archive",
});
