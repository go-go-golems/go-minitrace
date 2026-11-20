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
function tokenProvenance(filters) {
  const mt = require("minitrace");
  const db = _dbFrom([filters.path]);
  try {
    const recipe = mt.query().TokenUsageRows().Build();
    const r = db.execute(recipe);
    if (r.error) return { error: r.error };
    const cols = r.columns;
    const rows = r.rows;
    const wanted = ["token_origin", "token_unit", "token_method", "estimated_tokens", "reported_usage_fields", "input_tokens", "output_tokens", "total_tokens", "tool_id", "turn_index"];
    const slim = rows.map((row) => {
      const o = {};
      for (const f of wanted) o[f] = row[f] === undefined ? null : row[f];
      return o;
    });
    // provenance breakdown without summing incompatible units
    const byOrigin = {};
    for (const row of slim) {
      const k = String(row.token_origin);
      byOrigin[k] = (byOrigin[k] || 0) + 1;
      if (!byOrigin[k + "_units"]) byOrigin[k + "_units"] = [];
      if (!byOrigin[k + "_units"].includes(row.token_unit)) byOrigin[k + "_units"].push(row.token_unit);
    }
    const estRows = rows.filter((x) => x.estimated_tokens !== null && x.estimated_tokens !== undefined).slice(0, 5);
    const toolRows = rows.filter((x) => x.tool_call_id).slice(0, 5);
    return { columns: cols, rows: slim.slice(0, 20), row_count: rows.length, provenance_counts: byOrigin, estimate_rows: estRows, tool_level_rows: toolRows };
  } finally { db.close(); }
}
__section__("filters", { title: "Filters", fields: { path: { type: "string", help: "path" } } });
__verb__("tokenProvenance", { name: "token-provenance", short: "token-provenance", fields: { filters: { bind: "filters" } } });
