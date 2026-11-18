const completeRows = function(result) {
  if (result.error || !result.complete) {
    throw new Error(result.error || "Incomplete query result");
  }
  return result.rows;
};

__section__("filters", {
  title: "Filters",
  fields: {
    working_directory_like: { type: "string", help: "Substring match for working directory" },
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
  const cwd = filters.working_directory_like ? "%" + filters.working_directory_like + "%" : null;
  return _withDB(mt, (db) => {

  const workspaceRows = completeRows(db.execute(mt.sql`
    SELECT COALESCE(working_directory, '(none)') AS working_directory, COUNT(*) AS session_count, AVG(tool_call_count) AS avg_tool_calls
    FROM sessions
    WHERE (${cwd} IS NULL OR COALESCE(working_directory, '') LIKE ${cwd})
    GROUP BY 1
    ORDER BY session_count DESC, avg_tool_calls DESC, working_directory ASC
    LIMIT ${filters.limit}
  `.build()));

  const highlightRows = completeRows(db.execute(mt.sql`
    SELECT COALESCE(working_directory, '(none)') AS working_directory, session_id AS id, title, tool_call_count
    FROM sessions
    WHERE (${cwd} IS NULL OR COALESCE(working_directory, '') LIKE ${cwd})
    ORDER BY tool_call_count DESC, session_id ASC
  `.build()));

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
  });
}

__verb__("workspaceScoreboard", {
  name: "workspace-scoreboard",
  short: "Score workspaces using normalized mt.db tables",
  fields: { filters: { bind: "filters" } }
});
