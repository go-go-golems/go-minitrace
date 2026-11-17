function allNine() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    const result = db.execute(mt.query().SessionToolCounts().Build());
    if (result.error || !result.complete) {
      throw new Error(result.error || "incomplete tool counts");
    }
    const turnRows = db.query(`
      SELECT s.session_id AS session_id,
             (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS actual_turns
      FROM sessions s
      ORDER BY s.session_id
    `);
    const bySession = {};
    for (const row of turnRows) {
      bySession[row.session_id] = row.actual_turns;
    }
    return result.rows.map((row) => ({
      session_id: row.session_id,
      actual_turns: bySession[row.session_id] ?? null,
      actual_tools: row.tool_count,
    }));
  } finally {
    db.close();
  }
}

__verb__("allNine", {
  name: "all-nine",
  short: "Count actual turns and tool rows for all loaded sessions",
  fields: {},
});
