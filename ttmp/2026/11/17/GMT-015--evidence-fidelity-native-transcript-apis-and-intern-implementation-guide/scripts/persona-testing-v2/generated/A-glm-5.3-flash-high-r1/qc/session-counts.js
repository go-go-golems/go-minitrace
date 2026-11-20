// Real-corpus session counts via the normalized SQLite engine (JS path).
function sessionCounts() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    const result = db.execute({
      sql: `
        SELECT s.session_id, s.agent_framework,
          (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_count,
          (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_count
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

__verb__("sessionCounts", {
  name: "session-counts",
  short: "Count turns and tool_calls rows per real normalized session",
});
