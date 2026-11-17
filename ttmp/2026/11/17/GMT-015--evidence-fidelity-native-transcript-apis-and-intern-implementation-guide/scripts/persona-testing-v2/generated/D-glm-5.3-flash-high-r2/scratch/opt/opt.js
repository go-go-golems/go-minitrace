function opt(args) {
  const mt = require("minitrace");
  const db = mt.db().File(args.source).SQLiteMemory().MaxRows(100000).MaxCellBytes(4000000).Build();
  try {
    const tables = db.tables();
    const forbidden = db.queryResult("DELETE FROM sessions");
    const strict = (() => {
      const sdb = mt.db().File(args.source).SQLiteMemory().MaxCellBytes(16).StrictEvidence(true).Build();
      try { return sdb.queryResult("SELECT session_id FROM sessions LIMIT 1"); }
      finally { sdb.close(); }
    })();
    return [{ tables: tables.join("|"), forbidden_error: forbidden.error || null, strict_error: strict.error || null, strict_complete: strict.complete }];
  } finally { db.close(); }
}
__verb__("opt", { name: "opt", short: "Optional probes: tables, forbidden SQL, strict evidence", fields: { args: { bind: "opt_args" } } });
__section__("opt_args", { title: "Source", fields: { source: { type: "string", required: true, help: "archive path" } } });
