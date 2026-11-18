const mt = require("minitrace");

function report() {
  const sources = mt.sources().runtimeArchives().build();
  const db = mt.db().sources(sources)
    .import({autoConvert: true, strict: true})
    .limits({maxRows: 2000, maxCellBytes: 8192, strict: true})
    .build();
  try {
    const minimum = mt.param.integer("minimum").default(0)
      .ui({widget: "number", label: "Minimum tool calls", help: "Keep sessions with at least this many physical tool_calls"});
    const plan = mt.sql`
      SELECT s.session_id AS session_id, COUNT(t.session_id) AS tool_count
      FROM sessions s
      LEFT JOIN tool_calls t ON t.session_id = s.session_id
      GROUP BY s.session_id
      HAVING COUNT(t.session_id) >= ${minimum}
      ORDER BY s.session_id
    `
      .columns({session_id: mt.type.string(), tool_count: mt.type.integer()})
      .mapRows(row => ({
        session_id: row.session_id,
        label: row.tool_count + " tools",
      }), {
        columns: {session_id: mt.type.string(), label: mt.type.string()},
      })
      .build();
    const result = db.execute(plan, {params: {minimum: 0}});
    if (result.error || !result.complete) {
      throw Error(result.error || "Incomplete result");
    }
    return result.rows.map(r => ({session_id: r.session_id, label: r.label}));
  } finally {
    db.close();
  }
}

__verb__("report", {name: "report"});
