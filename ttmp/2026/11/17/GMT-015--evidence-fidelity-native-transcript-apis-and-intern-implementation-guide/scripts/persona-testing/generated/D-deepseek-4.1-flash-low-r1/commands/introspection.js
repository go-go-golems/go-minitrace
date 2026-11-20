// introspection.js — use the supported JS schema introspection APIs.
// db.tables() and db.schema() are the approved alternatives to querying
// sqlite_master directly.

function tables() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.tables();
  } finally {
    db.close();
  }
}

function schema() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.schema();
  } finally {
    db.close();
  }
}

__verb__("tables", {
  name: "tables",
  short: "List normalized SQLite tables through db.tables()",
});

__verb__("schema", {
  name: "schema",
  short: "Describe the normalized SQLite schema through db.schema()",
});
