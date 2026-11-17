function perSessionCounts(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query(`
      SELECT
        s.session_id AS session_id,
        s.agent_framework AS framework,
        (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_count,
        (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_count,
        s.started_at AS started_at,
        s.ended_at AS ended_at
      FROM sessions s
      ORDER BY s.agent_framework, s.started_at
    `);
  } finally {
    db.close();
  }
}

__verb__("perSessionCounts", {
  name: "per-session",
  short: "Per-session actual turn and tool row counts",
  fields: {},
});
