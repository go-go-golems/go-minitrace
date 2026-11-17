function _src(paths) {
  const mt = require("minitrace");
  const s = mt.sources();
  for (const p of paths) s.File(p);
  return s.Build();
}
function _dbFrom(paths, extra) {
  const mt = require("minitrace");
  let db = mt.db().Sources(_src(paths)).SQLiteMemory();
  if (extra) db = extra(db);
  return db.Build();
}
function toolFidelity(filters) {
  const mt = require("minitrace");
  const db = _dbFrom([filters.path]);
  try {
    const recipe = mt.query().ToolRows().Build();
    const base = db.execute(recipe);
    if (base.error) return { error: base.error };
    const cols = base.columns;
    const rows = base.rows;
    const fields = ["tool_call_id", "source_ordinal", "timestamp", "arguments_json", "output_status", "outcome_status", "success", "turn_index"];
    const slim = rows.map((r) => {
      const o = {};
      for (const f of fields) o[f] = r[f] === undefined ? null : r[f];
      return o;
    });
    const fieldIdx = {};
    for (const f of fields) fieldIdx[f] = cols.indexOf(f);
    // archive order vs chronological order
    const byArchive = rows.slice().sort((a, b) => a.source_ordinal - b.source_ordinal).map((r) => r.tool_call_id);
    const byChrono = rows.slice().sort((a, b) => String(a.timestamp).localeCompare(String(b.timestamp))).map((r) => r.tool_call_id);
    const ordinalsArchiveOrder = rows.slice().sort((a, b) => a.source_ordinal - b.source_ordinal).map((r) => r.source_ordinal);
    const statusPairs = {};
    for (const r of rows) {
      const k = String(r.output_status) + "|" + String(r.outcome_status);
      statusPairs[k] = (statusPairs[k] || 0) + 1;
    }
    return {
      columns: cols, field_column_index: fieldIdx,
      rows: slim.slice(0, 12), row_count: rows.length,
      archive_order_source_ordinal: byArchive,
      chronological_order_timestamp: byChrono,
      order_differs: JSON.stringify(byArchive) !== JSON.stringify(byChrono),
      output_status_vs_outcome_status_pairs: statusPairs,
      source_ordinals_archive_order: ordinalsArchiveOrder,
    };
  } finally { db.close(); }
}
__section__("filters", { title: "Filters", fields: { path: { type: "string", help: "path" } } });
__verb__("toolFidelity", { name: "tool-fidelity", short: "tool-fidelity", fields: { filters: { bind: "filters" } } });
