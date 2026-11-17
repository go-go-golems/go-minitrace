// Task A: count actual turns and tool_calls rows per session across all loaded archives.
// Run via: go-minitrace query commands counts per-session-counts --archive-glob '...'

function perSessionCounts() {
  const mt = require("minitrace");
  const db = mt.db()
    .RuntimeArchives()
    .SQLiteMemory()
    .MaxCellBytes(16000)
    .StrictEvidence(true)
    .Build();
  try {
    const result = db.execute({
      sql: `
        SELECT s.session_id,
               s.agent_framework AS framework,
               (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_rows,
               (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_rows,
               s.turn_count AS metric_turn_count,
               s.tool_call_count AS metric_tool_call_count
        FROM sessions s
        ORDER BY s.agent_framework, s.session_id
      `,
      args: [],
    });
    if (result.error || !result.complete) {
      throw new Error(result.error || "Incomplete evidence");
    }
    return result.rows;
  } finally {
    db.close();
  }
}

__verb__("perSessionCounts", {
  name: "per-session-counts",
  short: "Count actual turns and tool_calls rows per session",
});
