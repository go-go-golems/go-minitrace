__section__("inputs", { title: "Inputs", fields: {} });
function schemaProbe() {
  const mt = require("minitrace");
  const f = "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/pi/active/2026-09/01a0a5c3-342f-7447-be74-b6c35f542ff7.minitrace.json";
  const db = mt.db().File(f).SQLiteMemory().Build();
  try {
    const cols = {};
    for (const t of ["sessions","metrics","tool_calls","turns"]) {
      const r = db.execute({sql: "SELECT * FROM " + t + " LIMIT 1"});
      cols[t] = {columns: r.columns, error: r.error ?? null, rows: r.rows.length};
    }
    const metrics = db.execute({sql: "SELECT * FROM metrics"});
    return [{tables: cols, metrics_rows: metrics.rows, metrics_error: metrics.error ?? null,
             tablesList: db.tables()}];
  } finally { db.close(); }
}
__verb__("schemaProbe", { name: "schema-probe", short: "Probe schema" });
