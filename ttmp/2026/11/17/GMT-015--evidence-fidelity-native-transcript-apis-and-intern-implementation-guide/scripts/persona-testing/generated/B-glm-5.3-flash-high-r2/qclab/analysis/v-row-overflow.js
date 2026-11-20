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
function rowOverflow() {
  const db = _dbFrom(["/private/tmp/gmt015-persona/inputs/archive.json"], (d) => d.MaxRows(1));
  try {
    const r = db.queryResult("SELECT 1 AS n UNION ALL SELECT 2 ORDER BY n");
    return {
      columns: r.columns, rows: r.rows, count: r.count,
      complete: r.complete, truncated: r.truncated || null,
      rowsTruncated: r.rowsTruncated === undefined ? null : r.rowsTruncated,
      error: r.error || null, cellDiagnostics: r.cellDiagnostics || null,
    };
  } finally { db.close(); }
}
__verb__("rowOverflow", { name: "row-overflow", short: "row-overflow" });
