// good.js — root query verb `good`.
// Reports per-session tool counts for the runtime archives supplied through
// the standard --archive-glob runtime flag. `source` is a required string
// filter over sessions.source_format (for example: claude-code, codex, pi).
//
// The SQL uses bound positional parameters (?) rather than string splicing,
// so the filter value never becomes SQL text.

__section__("filters", {
  title: "Source selection",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Required source_format substring, e.g. claude-code, codex, or pi",
    },
    limit: {
      type: "int",
      default: 100,
      help: "Maximum sessions to report",
    },
  },
});

function good(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    const rows = db.query(
      `SELECT
         s.session_id,
         s.agent_framework AS framework,
         s.source_format,
         s.tool_call_count AS declared_tool_calls,
         COUNT(tc.tool_call_id) AS actual_tool_calls,
         COUNT(DISTINCT tc.tool_name) AS distinct_tools
       FROM sessions s
       LEFT JOIN tool_calls tc ON tc.session_id = s.session_id
       WHERE s.source_format LIKE ?
       GROUP BY s.session_id, s.agent_framework, s.source_format, s.tool_call_count
       ORDER BY s.source_format, s.session_id
       LIMIT ?`,
      `%${filters.source}%`,
      filters.limit
    );

    // The command deliberately returns a plain JS array. The Glazed CLI
    // renders each element as one result row.
    return rows;
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Per-session tool counts for runtime archives, filtered by required source",
  fields: {
    filters: { bind: "filters" },
  },
});
