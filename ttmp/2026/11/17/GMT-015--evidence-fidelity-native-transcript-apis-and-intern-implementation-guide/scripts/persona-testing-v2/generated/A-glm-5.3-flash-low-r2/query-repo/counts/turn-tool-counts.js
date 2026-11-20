function sessionCounts() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query(`
      SELECT s.session_id, s.agent_framework,
        (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_rows,
        (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_rows
      FROM sessions s
      ORDER BY s.session_id
    `);
  } finally {
    db.close();
  }
}

__verb__("sessionCounts", {
  name: "session-counts",
  short: "Count turns and tool_calls rows per session via JS",
});
