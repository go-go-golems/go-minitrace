const cookbook = require("./lib/cookbook");

__section__("filters", {
  title: "Filters",
  fields: {
    framework: { type: "stringList", help: "Restrict to specific agent frameworks" },
    working_directory_like: { type: "string", help: "Substring match for working directory" },
    min_tool_calls: { type: "int", default: 0, help: "Minimum tool call count per session" },
    limit: { type: "int", default: 8, help: "Maximum number of rows to return" }
  }
});

function _sessionWhere(mt, filters) {
  const clauses = ["1=1"];
  if (filters.framework?.length) {
    clauses.push(`environment->>'agent_framework' IN (${mt.sql.stringIn(filters.framework)})`);
  }
  if (filters.working_directory_like) {
    clauses.push(`COALESCE(operational_context->>'working_directory', '') LIKE ${mt.sql.like(filters.working_directory_like)}`);
  }
  if (filters.min_tool_calls) {
    clauses.push(`CAST(metrics->>'tool_call_count' AS BIGINT) >= ${filters.min_tool_calls}`);
  }
  return clauses.join("\n    AND ");
}

function sessionShapeRanker(filters) {
  const mt = require("minitrace");
  const whereSql = _sessionWhere(mt, filters);

  const baseRows = mt.query(`
    SELECT
      id,
      title,
      COALESCE(operational_context->>'working_directory', '(none)') AS working_directory,
      environment->>'model' AS model,
      CAST(metrics->>'tool_call_count' AS BIGINT) AS tool_call_count,
      CAST(metrics->>'turn_count' AS BIGINT) AS turn_count,
      COALESCE(CAST(timing->>'active_duration_seconds' AS DOUBLE), 0) AS active_duration_seconds
    FROM ${mt.tableName}
    WHERE ${whereSql}
    ORDER BY tool_call_count DESC, turn_count DESC, id ASC
    LIMIT ${filters.limit}
  `);

  const roleRows = mt.query(`
    SELECT
      id,
      turn->>'role' AS role,
      COUNT(*) AS role_count
    FROM ${mt.tableName}, UNNEST(turns) AS t(turn)
    WHERE ${whereSql}
    GROUP BY 1, 2
    ORDER BY id ASC, role ASC
  `);

  const uniqueToolRows = mt.query(`
    SELECT
      id,
      COUNT(DISTINCT call->>'tool_name') AS unique_tools
    FROM ${mt.tableName}, UNNEST(tool_calls) AS t(call)
    WHERE ${whereSql}
      AND (call->>'tool_name') IS NOT NULL
    GROUP BY 1
  `);

  const rolesBySession = cookbook.groupBy(roleRows, "id");
  const uniqueToolsBySession = cookbook.firstBy(uniqueToolRows, "id");

  return baseRows.map((row, index) => {
    const roleRowsForSession = rolesBySession[row.id] || [];
    const userTurns = roleRowsForSession.find((r) => r.role === 'user')?.role_count || 0;
    const assistantTurns = roleRowsForSession.find((r) => r.role === 'assistant')?.role_count || 0;
    const uniqueTools = uniqueToolsBySession[row.id]?.unique_tools || 0;
    const shapeLabel = cookbook.classifySessionShape({
      tool_call_count: row.tool_call_count,
      turn_count: row.turn_count,
      unique_tools: uniqueTools,
      user_turns: userTurns,
      assistant_turns: assistantTurns,
    });
    const complexityScore = cookbook.round1(
      cookbook.safeNumber(row.tool_call_count) * 0.5 +
        cookbook.safeNumber(uniqueTools) * 2 +
        cookbook.safeNumber(row.turn_count) * 0.1 +
        cookbook.safeNumber(row.active_duration_seconds) / 900,
    );

    return {
      rank: index + 1,
      id: row.id,
      title: row.title,
      working_directory: row.working_directory,
      model: row.model,
      tool_call_count: row.tool_call_count,
      turn_count: row.turn_count,
      user_turns: userTurns,
      assistant_turns: assistantTurns,
      unique_tools: uniqueTools,
      shape_label: shapeLabel,
      complexity_score: complexityScore,
    };
  });
}

function sessionSpotlights(filters) {
  const mt = require("minitrace");
  const whereSql = _sessionWhere(mt, filters);

  const baseRows = mt.query(`
    SELECT
      id,
      title,
      COALESCE(operational_context->>'working_directory', '(none)') AS working_directory,
      CAST(metrics->>'tool_call_count' AS BIGINT) AS tool_call_count,
      CAST(metrics->>'turn_count' AS BIGINT) AS turn_count
    FROM ${mt.tableName}
    WHERE ${whereSql}
    ORDER BY tool_call_count DESC, turn_count DESC, id ASC
    LIMIT ${filters.limit}
  `);

  const toolRows = mt.query(`
    SELECT
      id,
      call->>'tool_name' AS tool_name,
      COUNT(*) AS tool_uses
    FROM ${mt.tableName}, UNNEST(tool_calls) AS t(call)
    WHERE ${whereSql}
      AND (call->>'tool_name') IS NOT NULL
    GROUP BY 1, 2
    ORDER BY id ASC, tool_uses DESC, tool_name ASC
  `);

  const roleRows = mt.query(`
    SELECT
      id,
      turn->>'role' AS role,
      COUNT(*) AS role_count
    FROM ${mt.tableName}, UNNEST(turns) AS t(turn)
    WHERE ${whereSql}
    GROUP BY 1, 2
    ORDER BY id ASC, role_count DESC, role ASC
  `);

  const toolsBySession = cookbook.groupBy(toolRows, "id");
  const rolesBySession = cookbook.groupBy(roleRows, "id");

  return baseRows.map((row) => ({
    id: row.id,
    title: row.title,
    working_directory: row.working_directory,
    workspace_slug: cookbook.shortWorkspace(row.working_directory),
    tool_call_count: row.tool_call_count,
    turn_count: row.turn_count,
    dominant_tools: cookbook.joinTopValues(toolsBySession[row.id] || [], "tool_name", 3),
    role_mix: (rolesBySession[row.id] || []).map((entry) => `${entry.role}:${entry.role_count}`).join(", "),
  }));
}

__verb__("sessionShapeRanker", {
  name: "session-shape-ranker",
  short: "Join session, turn, and tool aggregates to classify session shapes",
  fields: { filters: { bind: "filters" } }
});

__verb__("sessionSpotlights", {
  name: "session-spotlights",
  short: "Build spotlight cards from per-session tool and role summaries",
  fields: { filters: { bind: "filters" } }
});
