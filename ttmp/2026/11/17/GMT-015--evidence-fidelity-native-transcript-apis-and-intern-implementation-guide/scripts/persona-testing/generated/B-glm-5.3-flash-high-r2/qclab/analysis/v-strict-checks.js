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
function strictChecks() {
  const db = _dbFrom(["/private/tmp/gmt015-persona/inputs/archive.json"], (d) => d.MaxCellBytes(3).StrictEvidence(true));
  try {
    const strict = db.queryResult("SELECT 'abé' AS text");
    return {
      complete: strict.complete,
      error: strict.error || null,
      rows: strict.rows,
      count: strict.count,
      truncated: strict.truncated || null,
      cellDiagnostics: strict.cellDiagnostics || null,
    };
  } finally { db.close(); }
}
__verb__("strictChecks", { name: "strict-checks", short: "strict-checks" });
