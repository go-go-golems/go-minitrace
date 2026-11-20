function schema() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try {
    return db.schema();
  } finally {
    db.close();
  }
}
__verb__("schema", {
  name: "schema",
  short: "Introspect normalized schema",
  fields: {},
});
