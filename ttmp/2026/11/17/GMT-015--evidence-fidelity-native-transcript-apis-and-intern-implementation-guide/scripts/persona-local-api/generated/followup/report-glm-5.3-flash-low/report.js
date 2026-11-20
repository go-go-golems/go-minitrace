const mt = require("minitrace");

function report() {
  const sources = mt.sources().runtimeArchives().build();
  const db = mt.db().sources(sources)
    .import({ autoConvert: true, strict: true })
    .limits({ maxRows: 2000, maxCellBytes: 8192, strict: true })
    .build();
  try {
    const minimum = mt.param.integer("minimum").default(0);
    const plan = mt.sql`
      SELECT s.session_id AS session_id,
             (SELECT COUNT(*) FROM tool_calls t WHERE t.session_id = s.session_id) AS tool_count
      FROM sessions s
      GROUP BY s.session_id
      HAVING tool_count >= ${minimum}
      ORDER BY s.session_id
    `.columns({ session_id: mt.type.string(), tool_count: mt.type.integer() })
      .orderBy(row => row.session_id.asc())
      .mapRows(row => ({ session_id: row.session_id, label: `${row.tool_count} tools` }), {
        columns: { session_id: mt.type.string(), label: mt.type.string() },
      })
      .build();
    const result = db.execute(plan, { params: { minimum: 0 } });
    if (result.error || !result.complete) throw Error(result.error || "Incomplete result");
    return result.rows;
  } finally {
    db.close();
  }
}

__verb__("report", { name: "report" });
