const ARCHIVE = "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive.json";

function optional() {
  const mt = require("minitrace");
  const out = { archive: ARCHIVE };

  const db1 = mt.db().Archive(ARCHIVE).SQLiteMemory().MaxRows(1).Build();
  try {
    out.row_overflow = db1.execute({ sql: "SELECT tool_call_id, source_ordinal FROM tool_calls ORDER BY source_ordinal" });
  } finally { db1.close(); }

  const db2 = mt.db().Archive(ARCHIVE).SQLiteMemory().Build();
  try {
    out.invalid_envelope = db2.execute({ sql: "SELECT * FROM definitely_missing_table" });
    try {
      db2.query("SELECT * FROM definitely_missing_table");
      out.invalid_query_threw = false;
    } catch (e) {
      out.invalid_query_threw = true;
      out.invalid_query_error = String(e && e.message ? e.message : e);
    }
  } finally { db2.close(); }

  return out;
}

__verb__("optional", {
  name: "optional",
  short: "Optional overflow and invalid-query envelope probes",
  fields: {},
});
