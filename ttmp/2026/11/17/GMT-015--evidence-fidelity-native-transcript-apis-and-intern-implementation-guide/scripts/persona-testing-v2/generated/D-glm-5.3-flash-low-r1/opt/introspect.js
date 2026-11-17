__verb__("introspect", {
  name: "introspect",
  short: "Introspect schema and test forbidden SQL",
  fields: {
    source: { type: "string", required: true, help: "archive path" },
  },
});
function introspect(source) {
  const mt = require("minitrace");
  const db = mt.db().File(source).SQLiteMemory().Build();
  try {
    const tables = db.tables();
    const schema = db.schema();
    let forbidden = null;
    try { db.query("DROP TABLE sessions"); } catch (e) { forbidden = String(e && e.message || e); }
    const res = db.queryResult ? null : null;
    return { tables: tables, schema_sample: schema && schema.slice ? schema.slice(0, 4) : schema, forbidden_result: forbidden };
  } finally { db.close(); }
}
