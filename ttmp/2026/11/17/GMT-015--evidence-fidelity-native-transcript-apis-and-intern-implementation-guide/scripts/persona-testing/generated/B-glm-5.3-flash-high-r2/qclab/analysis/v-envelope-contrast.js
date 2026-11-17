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
function envelopeContrast(filters) {
  const mt = require("minitrace");
  const out = {};
  const db = _dbFrom(["/private/tmp/gmt015-persona/inputs/archive.json"]);
  try {
    const first = db.queryOne("SELECT session_id FROM sessions LIMIT 1");
    const sid = (filters && filters.session_id) || (first ? first.session_id : null);

    // builder recipe
    const recipe = mt.query().ToolRows().Build();
    try {
      const exec = db.execute(recipe);
      out.builder_recipe = {
        sql_head: recipe.sql().slice(0, 160),
        args: recipe.args(),
        count: exec.count,
        complete: exec.complete,
        error: exec.error || null,
      };
    } catch (e) {
      out.builder_recipe = { threw: String(e).slice(0, 200) };
    }

    // bound-argument SQL
    const bound = db.queryResult("SELECT session_id, agent_framework FROM sessions WHERE session_id = ?", sid);
    out.bound_sql = { sql: "SELECT session_id, agent_framework FROM sessions WHERE session_id = ?", arg: sid, rows: bound.rows, count: bound.count, complete: bound.complete, error: bound.error || null };

    // invalid SQL: envelope path
    const bad = db.queryResult("SELECT no_such_col FROM no_such_table");
    out.invalid_queryResult = { complete: bad.complete, error: bad.error || null, rows: bad.rows, count: bad.count };

    // invalid SQL: throwing convenience paths
    const threw = {};
    try { db.query("SELECT no_such_col FROM no_such_table"); threw.query = "no-throw"; }
    catch (e) { threw.query = String(e).slice(0, 200); }
    try { db.queryOne("SELECT no_such_col FROM no_such_table"); threw.queryOne = "no-throw"; }
    catch (e) { threw.queryOne = String(e).slice(0, 200); }
    out.throwing_paths = threw;
  } finally { db.close(); }
  return out;
}
__section__("filters", { title: "Filters", fields: { session: { type: "string", help: "session" } } });
__verb__("envelopeContrast", { name: "envelope-contrast", short: "envelope-contrast", fields: { filters: { bind: "filters" } } });
