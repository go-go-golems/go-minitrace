__section__("params", { fields: { files: { type: "stringList" } } });
function tables(params) {
  const mt = require("minitrace");
  const db = mt.db().Files(params.files).SQLiteMemory().Build();
  try { return { tables: db.tables() }; } finally { db.close(); }
}
__verb__("tables", { name: "tables", short: "list tables", fields: { params: { bind: "params" } } });
