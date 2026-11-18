const completeRows = function(result) {
  if (result.error || !result.complete) {
    throw new Error(result.error || "Incomplete query result");
  }
  return result.rows;
};

const {queryFilters} = require("../lib/query-filters");

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

function _db(mt) { return mt.db().sources(mt.sources().runtimeArchives().build())
    .import({autoConvert: true, strict: true}).build(); }

function _withDB(mt, fn) {
  const db = _db(mt);
  try {
    return fn(db);
  } finally {
    db.close();
  }
}

function workspaceScoreboard(filters) {
  const mt = require("minitrace");
  const {frameworkKeys, cwd, minTools} = queryFilters(filters);
  return _withDB(mt, (db) => {

  const workspaceRows = completeRows(db.execute(mt.sql`
    SELECT
      COALESCE(working_directory, '(none)') AS working_directory,
      COUNT(*) AS session_count,
      AVG(tool_call_count) AS avg_tool_calls,
      AVG(turn_count) AS avg_turns,
      MAX(started_at) AS latest_started_at
    FROM sessions
    WHERE (${frameworkKeys} IS NULL OR (agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(agent_framework) || '|') > 0))
    AND (${cwd} IS NULL OR COALESCE(working_directory, '') LIKE ${cwd})
    AND (${minTools} IS NULL OR tool_call_count >= ${minTools})
    GROUP BY 1
    ORDER BY session_count DESC, avg_tool_calls DESC, working_directory ASC
    LIMIT ${filters.limit}
  `.build()));

  const highlightRows = completeRows(db.execute(mt.sql`
    SELECT COALESCE(working_directory, '(none)') AS working_directory, session_id AS id, title, tool_call_count
    FROM sessions
    WHERE (${frameworkKeys} IS NULL OR (agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(agent_framework) || '|') > 0))
    AND (${cwd} IS NULL OR COALESCE(working_directory, '') LIKE ${cwd})
    AND (${minTools} IS NULL OR tool_call_count >= ${minTools})
    ORDER BY tool_call_count DESC, session_id ASC
  `.build()));

  const toolRows = completeRows(db.execute(mt.sql`
    SELECT COALESCE(s.working_directory, '(none)') AS working_directory, t.tool_name, COUNT(*) AS tool_uses
    FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
    WHERE (${frameworkKeys} IS NULL OR (s.agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(s.agent_framework) || '|') > 0))
    AND (${cwd} IS NULL OR COALESCE(s.working_directory, '') LIKE ${cwd})
    AND (${minTools} IS NULL OR s.tool_call_count >= ${minTools})
      AND t.tool_name IS NOT NULL
    GROUP BY 1, 2
    ORDER BY working_directory ASC, tool_uses DESC, tool_name ASC
  `.build()));

  const bestSessionByWorkspace = cookbook.firstBy(highlightRows, "working_directory");
  const topToolByWorkspace = cookbook.firstBy(toolRows, "working_directory");

  return workspaceRows.map((row, index) => {
    const bestSession = bestSessionByWorkspace[row.working_directory] || {};
    const topTool = topToolByWorkspace[row.working_directory] || {};
    return {
      rank: index + 1,
      working_directory: row.working_directory,
      workspace_slug: cookbook.shortWorkspace(row.working_directory),
      session_count: row.session_count,
      avg_tool_calls: cookbook.round1(row.avg_tool_calls),
      avg_turns: cookbook.round1(row.avg_turns),
      latest_started_at: row.latest_started_at,
      focus_score: cookbook.round1(cookbook.safeNumber(row.session_count) * 4 + cookbook.safeNumber(row.avg_tool_calls) * 0.3 + cookbook.safeNumber(row.avg_turns) * 0.05),
      top_tool: topTool.tool_name || "",
      top_tool_uses: topTool.tool_uses || 0,
      sample_session_id: bestSession.id || "",
      sample_title: bestSession.title || "",
    };
  });
  });
}

function workspaceSessionHighlights(filters) {
  const mt = require("minitrace");
  const {frameworkKeys, cwd, minTools} = queryFilters(filters);
  return _withDB(mt, (db) => {

  const workspaceRows = completeRows(db.execute(mt.sql`
    SELECT COALESCE(working_directory, '(none)') AS working_directory, COUNT(*) AS session_count
    FROM sessions
    WHERE (${frameworkKeys} IS NULL OR (agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(agent_framework) || '|') > 0))
    AND (${cwd} IS NULL OR COALESCE(working_directory, '') LIKE ${cwd})
    AND (${minTools} IS NULL OR tool_call_count >= ${minTools})
    GROUP BY 1
    ORDER BY session_count DESC, working_directory ASC
    LIMIT ${filters.limit}
  `.build()));

  const sessionRows = completeRows(db.execute(mt.sql`
    SELECT COALESCE(working_directory, '(none)') AS working_directory, session_id AS id, title, tool_call_count, model
    FROM sessions
    WHERE (${frameworkKeys} IS NULL OR (agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(agent_framework) || '|') > 0))
    AND (${cwd} IS NULL OR COALESCE(working_directory, '') LIKE ${cwd})
    AND (${minTools} IS NULL OR tool_call_count >= ${minTools})
    ORDER BY working_directory ASC, tool_call_count DESC, session_id ASC
  `.build()));

  const toolRows = completeRows(db.execute(mt.sql`
    SELECT COALESCE(s.working_directory, '(none)') AS working_directory, t.tool_name, COUNT(*) AS tool_uses
    FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
    GROUP BY 1, 2
    ORDER BY working_directory ASC, tool_uses DESC, tool_name ASC
  `.build()));

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
  });
}

__verb__("workspaceScoreboard", {
  name: "workspace-scoreboard",
  short: "Combine normalized DB queries into a scored workspace leaderboard",
  fields: { filters: { bind: "filters" } }
});

__verb__("workspaceSessionHighlights", {
  name: "workspace-session-highlights",
  short: "Join workspace summaries with top session titles and dominant tools",
  fields: { filters: { bind: "filters" } }
});
