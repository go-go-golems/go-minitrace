// disallowed.js — intentional disallowed SQL introspection query.
// The sandbox allowlist rejects sqlite_master; the error should point the
// caller at db.schema()/db.tables(). We use queryResult so the error envelope
// is visible instead of being confused with an empty result set.

function disallowed() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    const result = db.queryResult(
      "SELECT name, type FROM sqlite_master ORDER BY name LIMIT 5"
    );
    return {
      error: result.error || "",
      count: result.count,
      complete: result.complete,
      rows: result.rows,
    };
  } finally {
    db.close();
  }
}

__verb__("disallowed", {
  name: "disallowed",
  short: "Intentionally query sqlite_master to show the sandbox diagnostic",
});
