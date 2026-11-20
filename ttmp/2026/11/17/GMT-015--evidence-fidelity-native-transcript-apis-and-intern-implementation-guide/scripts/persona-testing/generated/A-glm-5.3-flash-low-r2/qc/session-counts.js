// Per-session actual turn and tool-call row counts (not metadata aggregates).
__section__("filters", { title: "Filters", fields: { limit: { type: "int", default: 100, help: "Max rows" } } });

function sessionCounts(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query(`
      SELECT
        s.session_id,
        s.agent_framework AS framework,
        (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_count_actual,
        (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_count_actual,
        s.turn_count AS turn_count_meta,
        s.tool_call_count AS tool_count_meta
      FROM sessions s
      ORDER BY s.started_at
      LIMIT 100
    `);
  } finally { db.close?.(); }
}
