// Runnable minitrace JS command: per-session actual turn and tool-row counts.
// Run with:
//   go-minitrace query commands session-counts \
//     --query-repository <this-dir> \
//     --archive-glob '/path/*/active/*/*.minitrace.json' --output json
function sessionCounts() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query(`
      SELECT
        s.session_id AS session_id,
        s.agent_framework AS framework,
        (SELECT COUNT(*) FROM turns t
           WHERE t.session_id = s.session_id) AS turn_count,
        (SELECT COUNT(*) FROM tool_calls tc
           WHERE tc.session_id = s.session_id) AS tool_count
      FROM sessions s
      ORDER BY s.agent_framework, s.session_id
    `);
  } finally {
    db.close();
  }
}

__verb__("sessionCounts", {
  name: "session-counts",
  short: "Count actual turns and tool rows per session",
  fields: {}
});
