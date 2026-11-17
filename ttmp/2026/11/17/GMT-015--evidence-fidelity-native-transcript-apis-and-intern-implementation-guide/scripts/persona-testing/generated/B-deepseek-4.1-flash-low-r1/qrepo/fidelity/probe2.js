function tables(f) {
  const mt = require("minitrace");
  const db = mt.db().Files(["/private/tmp/gmt015-persona/inputs/archive.json"]).SQLiteMemory().Build();
  try { return db.tables().map(t => ({t})); } finally { db.close(); }
}
function tokenCols(f) {
  const mt = require("minitrace");
  const db = mt.db().Files(["/private/tmp/gmt015-persona/inputs/archive.json"]).SQLiteMemory().Build();
  try {
    const s = db.schema();
    return [{schema: JSON.stringify(s).slice(0, 6000)}];
  } finally { db.close(); }
}
__verb__("tables", { name: "tables", short: "tables", fields: {} });
__verb__("tokenCols", { name: "tokencols", short: "tokencols", fields: {} });
