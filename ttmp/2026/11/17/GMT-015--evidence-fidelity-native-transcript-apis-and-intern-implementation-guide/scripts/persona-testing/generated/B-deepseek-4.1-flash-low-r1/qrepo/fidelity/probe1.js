function hello(filters) {
  const mt = require("minitrace");
  const db = mt.db().Files(["/private/tmp/gmt015-persona/inputs/archive.json"]).SQLiteMemory().MaxCellBytes(3).Build();
  try {
    const len = mt.utf8ByteLength("abé");
    const permissive = db.queryResult("SELECT 'abé' AS text");
    const strictDb = mt.db().Files(["/private/tmp/gmt015-persona/inputs/archive.json"]).SQLiteMemory().MaxCellBytes(3).StrictEvidence(true).Build();
    let strict;
    try { strict = strictDb.queryResult("SELECT 'abé' AS text"); } catch (e) { strict = {threw: String(e)}; }
    finally { strictDb.close(); }
    return [{len, permissive: JSON.stringify(permissive), strict: JSON.stringify(strict)}];
  } finally { db.close(); }
}
__verb__("hello", { name: "hello", short: "probe", fields: {} });
