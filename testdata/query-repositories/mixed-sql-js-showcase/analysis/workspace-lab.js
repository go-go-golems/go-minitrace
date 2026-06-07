__section__("filters", {
  title: "Filters",
  fields: {
    working_directory_like: { type: "string", help: "Substring match for working directory" },
    limit: { type: "int", default: 8, help: "Maximum number of rows to return" }
  }
});

function _db(mt) { return mt.db().RuntimeArchives().Build(); }

function _workspaceWhere(mt, filters) {
  const clauses = ["1=1"];
  if (filters.working_directory_like) clauses.push(`COALESCE(working_directory, '') LIKE ${mt.sql.like(filters.working_directory_like)}`);
  return clauses.join("\n    AND ");
}

function workspaceScoreboard(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  const whereSql = _workspaceWhere(mt, filters);

  const workspaceRows = db.query(`
    SELECT COALESCE(working_directory, '(none)') AS working_directory, COUNT(*) AS session_count, AVG(tool_call_count) AS avg_tool_calls
    FROM sessions
    WHERE ${whereSql}
    GROUP BY 1
    ORDER BY session_count DESC, avg_tool_calls DESC, working_directory ASC
    LIMIT ${filters.limit}
  `);

  const highlightRows = db.query(`
    SELECT COALESCE(working_directory, '(none)') AS working_directory, session_id AS id, title, tool_call_count
    FROM sessions
    WHERE ${whereSql}
    ORDER BY tool_call_count DESC, session_id ASC
  `);

  const byWorkspace = {};
  for (const row of highlightRows) if (!byWorkspace[row.working_directory]) byWorkspace[row.working_directory] = row;

  return workspaceRows.map((row, index) => ({
    rank: index + 1,
    working_directory: row.working_directory,
    session_count: row.session_count,
    avg_tool_calls: Math.round(Number(row.avg_tool_calls || 0) * 10) / 10,
    sample_session_id: byWorkspace[row.working_directory]?.id || "",
    sample_title: byWorkspace[row.working_directory]?.title || "",
  }));
}

__verb__("workspaceScoreboard", {
  name: "workspace-scoreboard",
  short: "Score workspaces using normalized mt.db tables",
  fields: { filters: { bind: "filters" } }
});
