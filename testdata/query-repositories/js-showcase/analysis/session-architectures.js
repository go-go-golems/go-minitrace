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

function _db(mt) { return mt.db().RuntimeArchives().Build(); }

function _sessionWhere(mt, filters, alias) {
  const p = alias ? alias + "." : "";
  const clauses = ["1=1"];
  if (filters.framework?.length) clauses.push(`${p}agent_framework IN (${mt.sql.stringIn(filters.framework)})`);
  if (filters.working_directory_like) clauses.push(`COALESCE(${p}working_directory, '') LIKE ${mt.sql.like(filters.working_directory_like)}`);
  if (filters.min_tool_calls) clauses.push(`${p}tool_call_count >= ${filters.min_tool_calls}`);
  return clauses.join("\n    AND ");
}

function sessionShapeRanker(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  const whereSql = _sessionWhere(mt, filters, "s");

  const baseRows = db.query(`
    SELECT session_id AS id, title, COALESCE(working_directory, '(none)') AS working_directory, model, tool_call_count, turn_count, 0 AS active_duration_seconds
    FROM sessions s
    WHERE ${whereSql}
    ORDER BY tool_call_count DESC, turn_count DESC, session_id ASC
    LIMIT ${filters.limit}
  `);

  const roleRows = db.query(`
    SELECT t.session_id AS id, t.role, COUNT(*) AS role_count
    FROM turns t JOIN sessions s ON s.session_id = t.session_id
    WHERE ${whereSql}
    GROUP BY 1, 2
    ORDER BY id ASC, role ASC
  `);

  const uniqueToolRows = db.query(`
    SELECT t.session_id AS id, COUNT(DISTINCT t.tool_name) AS unique_tools
    FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
    WHERE ${whereSql} AND t.tool_name IS NOT NULL
    GROUP BY 1
  `);

  const rolesBySession = cookbook.groupBy(roleRows, "id");
  const uniqueToolsBySession = cookbook.firstBy(uniqueToolRows, "id");

  return baseRows.map((row, index) => {
    const roleRowsForSession = rolesBySession[row.id] || [];
    const userTurns = roleRowsForSession.find((r) => r.role === "user")?.role_count || 0;
    const assistantTurns = roleRowsForSession.find((r) => r.role === "assistant")?.role_count || 0;
    const uniqueTools = uniqueToolsBySession[row.id]?.unique_tools || 0;
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
      shape_label: cookbook.classifySessionShape({ tool_call_count: row.tool_call_count, turn_count: row.turn_count, unique_tools: uniqueTools, user_turns: userTurns, assistant_turns: assistantTurns }),
      complexity_score: cookbook.round1(cookbook.safeNumber(row.tool_call_count) * 0.5 + cookbook.safeNumber(uniqueTools) * 2 + cookbook.safeNumber(row.turn_count) * 0.1),
    };
  });
}

function sessionSpotlights(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  const whereSql = _sessionWhere(mt, filters, "s");

  const baseRows = db.query(`
    SELECT session_id AS id, title, COALESCE(working_directory, '(none)') AS working_directory, tool_call_count, turn_count
    FROM sessions s
    WHERE ${whereSql}
    ORDER BY tool_call_count DESC, turn_count DESC, session_id ASC
    LIMIT ${filters.limit}
  `);

  const toolRows = db.query(`
    SELECT t.session_id AS id, t.tool_name, COUNT(*) AS tool_uses
    FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
    WHERE ${whereSql} AND t.tool_name IS NOT NULL
    GROUP BY 1, 2
    ORDER BY id ASC, tool_uses DESC, tool_name ASC
  `);

  const roleRows = db.query(`
    SELECT t.session_id AS id, t.role, COUNT(*) AS role_count
    FROM turns t JOIN sessions s ON s.session_id = t.session_id
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
