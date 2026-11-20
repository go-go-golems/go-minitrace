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
function frameworkSummary(filters) {
  const db = _dbFrom([filters.path]);
  try {
    const sess = db.queryOne("SELECT session_id, agent_framework, model, turn_count, tool_call_count FROM sessions LIMIT 1") || {};
    const tools = db.queryOne("SELECT COUNT(*) AS n FROM tool_calls") || {};
    const mt = require("minitrace");
    const tokRecipe = mt.query().TokenUsageRows().Build();
    const tokRes = db.execute(tokRecipe);
    const byOrigin = {};
    if (!tokRes.error) {
      for (const r of tokRes.rows) {
        const k = r.token_origin + "/" + r.token_unit;
        byOrigin[k] = (byOrigin[k] || 0) + 1;
      }
    }
    const cache = db.cacheInfo ? db.cacheInfo() : null;
    return { session: sess, tool_rows: tools.n, token_breakdown_by_origin_unit: byOrigin,
      token_error: tokRes.error || null, cacheInfo: cache };
  } finally { db.close(); }
}
__section__("filters", { title: "Filters", fields: { path: { type: "string", help: "path" } } });
__verb__("frameworkSummary", { name: "framework-summary", short: "framework-summary", fields: { filters: { bind: "filters" } } });
