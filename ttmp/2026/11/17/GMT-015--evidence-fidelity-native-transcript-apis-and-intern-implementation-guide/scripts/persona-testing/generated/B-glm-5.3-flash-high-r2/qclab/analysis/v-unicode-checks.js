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
function unicodeChecks() {
  const mt = require("minitrace");
  const out = { utf8ByteLength_abe: mt.utf8ByteLength("abé") };
  const db = _dbFrom(["/private/tmp/gmt015-persona/inputs/archive.json"], (d) => d.MaxCellBytes(3));
  try {
    const permissive = db.queryResult("SELECT 'abé' AS text");
    out.permissive = {
      columns: permissive.columns,
      rows: permissive.rows,
      count: permissive.count,
      complete: permissive.complete,
      truncated: permissive.truncated,
      error: permissive.error || null,
      cellDiagnostics: permissive.cellDiagnostics || null,
    };
  } finally { db.close(); }
  return out;
}
__verb__("unicodeChecks", { name: "unicode-checks", short: "unicode-checks" });
