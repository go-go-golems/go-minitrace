__section__("session-counts", {
  title: "Per-session turn and tool counts from normalized tables",
});

function sessionCounts() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  return db.query(`
    SELECT s.session_id, s.agent_framework,
           COUNT(DISTINCT t.turn_index) AS turn_count,
           (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id=s.session_id) AS tool_count,
           MIN(t.timestamp) AS first_ts, MAX(t.timestamp) AS last_ts
    FROM sessions s JOIN turns t ON t.session_id=s.session_id
    GROUP BY s.session_id ORDER BY first_ts
  `);
}
