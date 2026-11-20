__section__("inputs", { title: "Inputs", fields: {} });
function toolrowsDiag() {
  const mt = require("minitrace");
  const db = mt.db().File("/private/tmp/gmt015-persona-v2-matrix/fixtures/archive.json").SQLiteMemory().Build();
  try {
    const r = db.execute(mt.query().ToolRows().Build());
    return [{complete: r.complete ?? null, error: r.error ?? null, truncated: r.truncated ?? null,
      rowsTruncated: r.rowsTruncated ?? null, count: r.count ?? null,
      cellDiagnostics: r.cellDiagnostics ?? null,
      rows: r.rows.map(x => ({id: x.tool_call_id, ord: x.source_ordinal, success: x.success,
        outcome_status: x.outcome_status, output_status: x.output_status,
        result: x.result, arguments_json: x.arguments_json}))}];
  } finally { db.close(); }
}
__verb__("toolrowsDiag", { name: "toolrows-diag", short: "ToolRows diagnostics" });
