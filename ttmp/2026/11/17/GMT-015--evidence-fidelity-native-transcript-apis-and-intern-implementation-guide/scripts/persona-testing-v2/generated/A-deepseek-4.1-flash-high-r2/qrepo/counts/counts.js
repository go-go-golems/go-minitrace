__section__("filters", {
  title: "Filters",
  fields: {
    limit: {
      type: "int",
      default: 100,
      help: "Maximum number of rows",
    },
  },
});

function counts(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query(`
      SELECT
        s.session_id AS session_id,
        s.agent_framework AS framework,
        (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_rows,
        (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_rows,
        s.turn_count AS sessions_rollup_turns,
        s.tool_call_count AS sessions_rollup_tools
      FROM sessions s
      ORDER BY s.session_id
      LIMIT ${Number(filters.limit) || 100}
    `);
  } finally {
    db.close();
  }
}

__verb__("counts", {
  name: "counts",
  short: "Count turns and tool calls per session from normalized tables",
  fields: {
    filters: { bind: "filters" }
  }
});
