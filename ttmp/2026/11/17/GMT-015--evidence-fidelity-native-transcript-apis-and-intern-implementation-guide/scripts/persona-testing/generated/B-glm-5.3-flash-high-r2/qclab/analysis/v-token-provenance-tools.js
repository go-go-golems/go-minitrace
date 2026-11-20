// Token provenance at tool scope: estimated output tokens use a DIFFERENT unit
// (estimated_tokens) and method (round_utf8_bytes_div_4) than recorded turn usage.
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
function tokenProvenanceTools(filters) {
  const mt = require("minitrace");
  const db = _dbFrom([filters.path]);
  try {
    const recipe = mt.query().TokenUsageRows().ByTool().Build();
    const r = db.execute(recipe);
    if (r.error) return { error: r.error };
    const rows = r.rows;
    const est = rows.filter((x) => x.estimated_tokens !== null && x.estimated_tokens !== undefined);
    const nonEst = rows.filter((x) => x.estimated_tokens == null);
    const byOrigin = {};
    for (const x of rows) {
      const k = String(x.token_origin);
      byOrigin[k] = (byOrigin[k] || 0) + 1;
      if (!byOrigin[k + "_units"]) byOrigin[k + "_units"] = [];
      if (!byOrigin[k + "_units"].includes(x.token_unit)) byOrigin[k + "_units"].push(x.token_unit);
    }
    return {
      row_count: rows.length,
      provenance_counts: byOrigin,
      estimate_samples: est.slice(0, 4),
      non_estimate_count: nonEst.length,
      note: "estimated rows use token_unit=estimated_tokens; must not be summed with token_unit=tokens",
    };
  } finally { db.close(); }
}
__section__("filters", { title: "Filters", fields: { path: { type: "string", help: "archive path" } } });
__verb__("tokenProvenanceTools", { name: "token-provenance-tools", short: "tool-scope estimated token provenance", fields: { filters: { bind: "filters" } } });
