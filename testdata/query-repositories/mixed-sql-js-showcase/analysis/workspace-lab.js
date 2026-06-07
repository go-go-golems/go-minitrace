const cookbook = require("./lib/cookbook");

__section__("filters", {
  title: "Filters",
  fields: {
    framework: { type: "stringList", help: "Restrict to specific agent frameworks" },
    min_tool_calls: { type: "int", default: 0, help: "Minimum tool-call count per session" },
    limit: { type: "int", default: 8, help: "Maximum number of rows to return" }
  }
});

function _workspaceWhere(mt, filters) {
  // Parenthesize DuckDB JSON-arrow predicates. The -> / ->> operators have low
  // precedence, so the wrapped form is easier to read and less brittle.
  const clauses = ["1=1"];
  if (filters.framework?.length) {
    clauses.push(`(environment->>'agent_framework') IN (${mt.sql.stringIn(filters.framework)})`);
  }
  if (filters.min_tool_calls) {
    clauses.push(`CAST(metrics->>'tool_call_count' AS BIGINT) >= ${filters.min_tool_calls}`);
  }
  return clauses.join("\n    AND ");
}

function workspaceScoreboard(filters) {
  const mt = require("minitrace");
  const whereSql = _workspaceWhere(mt, filters);

  const workspaceRows = mt.legacy.query(`
    SELECT
      COALESCE(operational_context->>'working_directory', '(none)') AS working_directory,
      COUNT(*) AS session_count,
      AVG(CAST(metrics->>'tool_call_count' AS DOUBLE)) AS avg_tool_calls,
      AVG(CAST(metrics->>'turn_count' AS DOUBLE)) AS avg_turns,
      MAX(timing->>'started_at') AS latest_started_at
    FROM ${mt.legacy.tableName}
    WHERE ${whereSql}
    GROUP BY 1
    ORDER BY session_count DESC, avg_tool_calls DESC, working_directory ASC
    LIMIT ${filters.limit}
  `);

  const highlightRows = mt.legacy.query(`
    SELECT
      COALESCE(operational_context->>'working_directory', '(none)') AS working_directory,
      id,
      title,
      CAST(metrics->>'tool_call_count' AS BIGINT) AS tool_call_count
    FROM ${mt.legacy.tableName}
    WHERE ${whereSql}
    ORDER BY tool_call_count DESC, id ASC
  `);

  const bestSessionByWorkspace = cookbook.firstBy(highlightRows, "working_directory");
  return workspaceRows.map((row, index) => {
    const best = bestSessionByWorkspace[row.working_directory] || {};
    return {
      rank: index + 1,
      working_directory: row.working_directory,
      workspace_slug: cookbook.shortWorkspace(row.working_directory),
      session_count: row.session_count,
      avg_tool_calls: cookbook.round1(row.avg_tool_calls),
      avg_turns: cookbook.round1(row.avg_turns),
      latest_started_at: row.latest_started_at,
      sample_session_id: best.id || "",
      sample_title: best.title || "",
    };
  });
}

__verb__("workspaceScoreboard", {
  name: "workspace-scoreboard",
  short: "JS workspace leaderboard built from multiple queries",
  fields: { filters: { bind: "filters" } }
});
