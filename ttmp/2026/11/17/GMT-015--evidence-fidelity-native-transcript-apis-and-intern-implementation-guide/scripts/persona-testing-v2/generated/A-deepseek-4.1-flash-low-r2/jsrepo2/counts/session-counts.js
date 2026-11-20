// Runnable minitrace JS query command: per-session actual turns/tools counts.
__section__("filters", {
  title: "Filters",
  fields: {},
});

function sessionCounts(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().MaxCellBytes(16000).Build();
  try {
    const ids = db.execute({ sql: "SELECT session_id FROM sessions ORDER BY session_id" });
    const diag = { ids: { error: ids.error || null, complete: ids.complete, count: ids.count } };

    // Turn rows recipe: count actual turns table rows per session.
    const turnRes = db.execute(mt.query().TurnRows().Build());
    diag.turns = { error: turnRes.error || null, complete: turnRes.complete, count: turnRes.count, truncated: turnRes.truncated || null, rowsTruncated: turnRes.rowsTruncated === undefined ? null : turnRes.rowsTruncated, cellDiagnostics: turnRes.cellDiagnostics || null };
    const turnCounts = {};
    for (const r of turnRes.rows) {
      turnCounts[r.session_id] = (turnCounts[r.session_id] || 0) + 1;
    }

    // SessionToolCounts recipe: actual tool_calls rows, retains zero-tool sessions.
    const toolRes = db.execute(mt.query().SessionToolCounts().Build());
    diag.tools = { error: toolRes.error || null, complete: toolRes.complete, count: toolRes.count, truncated: toolRes.truncated || null };
    const toolCounts = {};
    for (const r of toolRes.rows) toolCounts[r.session_id] = r.tool_count;

    // Independent SQL cross-check of both counts.
    const sqlTurn = db.execute({ sql: "SELECT session_id, COUNT(*) AS c FROM turns GROUP BY session_id" });
    const sqlTool = db.execute({ sql: "SELECT session_id, COUNT(*) AS c FROM tool_calls GROUP BY session_id" });
    if (sqlTurn.error || sqlTool.error) throw new Error(sqlTurn.error || sqlTool.error);
    const sqlTurnBy = {}, sqlToolBy = {};
    for (const r of sqlTurn.rows) sqlTurnBy[r.session_id] = r.c;
    for (const r of sqlTool.rows) sqlToolBy[r.session_id] = r.c;

    return ids.rows.map(function (row) {
      const id = row.session_id;
      return {
        session_id: id,
        turn_count: turnCounts[id] || 0,
        tool_count: toolCounts[id] === undefined ? 0 : toolCounts[id],
        sql_turn_count: sqlTurnBy[id] || 0,
        sql_tool_count: sqlToolBy[id] || 0,
        turn_rows_recipe: turnRes.rows.length,
        tool_recipe_rows: toolRes.rows.length,
        diag: diag,
      };
    });
  } finally {
    db.close();
  }
}

__verb__("sessionCounts", {
  name: "session-counts",
  short: "Count actual turns and tool_calls rows per session",
  fields: {
    filters: { bind: "filters" },
  },
});
