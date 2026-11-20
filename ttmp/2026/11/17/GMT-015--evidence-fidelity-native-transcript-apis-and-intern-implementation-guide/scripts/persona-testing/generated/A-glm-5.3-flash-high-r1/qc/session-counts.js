// Per-session turn and tool-call counts computed from actual normalized rows.
function sessionCounts(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query(`
      SELECT
        s.session_id AS session_id,
        s.agent_framework AS framework,
        (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_count,
        (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_count
      FROM sessions s
      ORDER BY s.started_at
    `);
  } finally {
    db.close();
  }
}

__verb__("sessionCounts", {
  name: "session-counts",
  short: "Per-session turn and tool counts from actual rows",
});
