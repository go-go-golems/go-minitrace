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

function _workspaceWhere(mt, filters) {
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

function workspaceScoreboard(filters) {
  const mt = require("minitrace");
  const whereSql = _workspaceWhere(mt, filters);

  const workspaceRows = mt.query(`
    SELECT
      COALESCE(operational_context->>'working_directory', '(none)') AS working_directory,
      COUNT(*) AS session_count,
      AVG(CAST(metrics->>'tool_call_count' AS DOUBLE)) AS avg_tool_calls,
      AVG(CAST(metrics->>'turn_count' AS DOUBLE)) AS avg_turns,
      AVG(COALESCE(CAST(timing->>'active_duration_seconds' AS DOUBLE), 0)) AS avg_active_seconds,
      MAX(timing->>'started_at') AS latest_started_at
    FROM ${mt.tableName}
    WHERE ${whereSql}
    GROUP BY 1
    ORDER BY session_count DESC, avg_tool_calls DESC, working_directory ASC
    LIMIT ${filters.limit}
  `);

  const highlightRows = mt.query(`
    SELECT
      COALESCE(operational_context->>'working_directory', '(none)') AS working_directory,
      id,
      title,
      CAST(metrics->>'tool_call_count' AS BIGINT) AS tool_call_count
    FROM ${mt.tableName}
    WHERE ${whereSql}
    ORDER BY tool_call_count DESC, id ASC
  `);

  const toolRows = mt.query(`
    SELECT
      COALESCE(operational_context->>'working_directory', '(none)') AS working_directory,
      call->>'tool_name' AS tool_name,
      COUNT(*) AS tool_uses
    FROM ${mt.tableName}, UNNEST(tool_calls) AS t(call)
    WHERE ${whereSql}
      AND (call->>'tool_name') IS NOT NULL
    GROUP BY 1, 2
    ORDER BY working_directory ASC, tool_uses DESC, tool_name ASC
  `);

  const bestSessionByWorkspace = cookbook.firstBy(highlightRows, "working_directory");
  const topToolByWorkspace = cookbook.firstBy(toolRows, "working_directory");

  return workspaceRows.map((row, index) => {
    const bestSession = bestSessionByWorkspace[row.working_directory] || {};
    const topTool = topToolByWorkspace[row.working_directory] || {};
    const focusScore = cookbook.round1(
      cookbook.safeNumber(row.session_count) * 4 +
        cookbook.safeNumber(row.avg_tool_calls) * 0.3 +
        cookbook.safeNumber(row.avg_turns) * 0.05 +
        cookbook.safeNumber(row.avg_active_seconds) / 600,
    );

    return {
      rank: index + 1,
      working_directory: row.working_directory,
      workspace_slug: cookbook.shortWorkspace(row.working_directory),
      session_count: row.session_count,
      avg_tool_calls: cookbook.round1(row.avg_tool_calls),
      avg_turns: cookbook.round1(row.avg_turns),
      latest_started_at: row.latest_started_at,
      focus_score: focusScore,
      top_tool: topTool.tool_name || "",
      top_tool_uses: topTool.tool_uses || 0,
      sample_session_id: bestSession.id || "",
      sample_title: bestSession.title || "",
    };
  });
}

function workspaceSessionHighlights(filters) {
  const mt = require("minitrace");
  const whereSql = _workspaceWhere(mt, filters);

  const workspaceRows = mt.query(`
    SELECT
      COALESCE(operational_context->>'working_directory', '(none)') AS working_directory,
      COUNT(*) AS session_count
    FROM ${mt.tableName}
    WHERE ${whereSql}
    GROUP BY 1
    ORDER BY session_count DESC, working_directory ASC
    LIMIT ${filters.limit}
  `);

  const sessionRows = mt.query(`
    SELECT
      COALESCE(operational_context->>'working_directory', '(none)') AS working_directory,
      id,
      title,
      CAST(metrics->>'tool_call_count' AS BIGINT) AS tool_call_count,
      environment->>'model' AS model
    FROM ${mt.tableName}
    WHERE ${whereSql}
    ORDER BY working_directory ASC, tool_call_count DESC, id ASC
  `);

  const toolRows = mt.query(`
    SELECT
      COALESCE(operational_context->>'working_directory', '(none)') AS working_directory,
      call->>'tool_name' AS tool_name,
      COUNT(*) AS tool_uses
    FROM ${mt.tableName}, UNNEST(tool_calls) AS t(call)
    WHERE ${whereSql}
      AND (call->>'tool_name') IS NOT NULL
    GROUP BY 1, 2
    ORDER BY working_directory ASC, tool_uses DESC, tool_name ASC
  `);

  const sessionsByWorkspace = cookbook.groupBy(sessionRows, "working_directory");
  const toolsByWorkspace = cookbook.groupBy(toolRows, "working_directory");

  return workspaceRows.map((row) => {
    const sessions = sessionsByWorkspace[row.working_directory] || [];
    const tools = toolsByWorkspace[row.working_directory] || [];
    return {
      working_directory: row.working_directory,
      workspace_slug: cookbook.shortWorkspace(row.working_directory),
      session_count: row.session_count,
      headline_titles: cookbook.joinTopValues(sessions, "title", 3),
      headline_models: cookbook.joinTopValues(sessions, "model", 2),
      dominant_tools: cookbook.joinTopValues(tools, "tool_name", 3),
    };
  });
}

__verb__("workspaceScoreboard", {
  name: "workspace-scoreboard",
  short: "Combine multiple queries into a scored workspace leaderboard",
  fields: { filters: { bind: "filters" } }
});

__verb__("workspaceSessionHighlights", {
  name: "workspace-session-highlights",
  short: "Join workspace summaries with top session titles and dominant tools",
  fields: { filters: { bind: "filters" } }
});
