// per-session-counts: count actual turns and tool_calls rows per session
// Usage:
//   go-minitrace query commands per-session-counts counts --query-repository <this dir> --archive-glob '...'

__verb__("perSessionCounts", {
  name: "counts",
  short: "Count actual turns and tool_calls rows per session (not aggregate metadata)",
});

function perSessionCounts() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    const rows = db.query(`
      SELECT s.session_id AS session_id,
             s.agent_framework AS framework,
             s.started_at AS started_at,
             (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_count,
             (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_count,
             (SELECT tool_name FROM tool_calls tc WHERE tc.session_id = s.session_id
              GROUP BY tool_name ORDER BY COUNT(*) DESC LIMIT 1) AS top_tool
      FROM sessions s
      ORDER BY s.started_at
    `);
    return {
      date_range: {
        first: rows.length ? rows[0].started_at : null,
        last: rows.length ? rows[rows.length - 1].started_at : null,
      },
      sessions: rows,
    };
  } finally {
    db.close();
  }
}
