// Explicit zero usage vs missing usage on the synthetic fixture.
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
function tokenZeroVsMissing() {
  const mt = require("minitrace");
  const db = _dbFrom(["/private/tmp/gmt015-persona/inputs/archive.json"]);
  try {
    const recipe = mt.query().TokenUsageRows().ByTurn().Build();
    const r = db.execute(recipe);
    if (r.error) return { error: r.error };
    return {
      rows: r.rows,
      note: "turn 1 records explicit zeros (recorded_usage); turn 0 has no usage fields at all (unknown, zero-filled) — zeros are not evidence of usage",
    };
  } finally { db.close(); }
}
__verb__("tokenZeroVsMissing", { name: "token-zero-vs-missing", short: "explicit zero vs missing usage provenance" });
