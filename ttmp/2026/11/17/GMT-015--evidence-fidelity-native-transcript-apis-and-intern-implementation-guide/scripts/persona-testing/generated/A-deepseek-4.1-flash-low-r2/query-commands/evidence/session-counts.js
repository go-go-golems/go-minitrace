__package__({
  name: "session-counts",
  short: "Count actual turn and tool rows per session from normalized child tables",
});

function sessionCounts() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query(`
      SELECT
        s.session_id,
        s.agent_framework,
        (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_count,
        (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_count
      FROM sessions s
      ORDER BY s.agent_framework, s.started_at
    `);
  } finally {
    db.close();
  }
}

__verb__("sessionCounts", {
  name: "session-counts",
  short: "Per-session actual turn/tool row counts",
  fields: {},
});
