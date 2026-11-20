function counts() {
  const mt = require("minitrace");
  const db = mt.db().sources(mt.sources().runtimeArchives().build())
    .limits({strict: true}).build();
  try {
    const sessions = mt.from(db.table("sessions"));
    const tools = mt.from(db.table("tool_calls"));
    const query = sessions.select(s => ({
      session_id: s.session_id,
      tool_count: tools.where(t => t.session_id.eq(s.session_id)).count(),
    })).orderBy(s => s.session_id.asc());
    const result = db.execute(query.build());
    if (result.error || !result.complete) throw new Error(result.error || "Incomplete evidence");
    return result.rows;
  } finally { db.close(); }
}
__verb__("counts", {name: "counts", short: "Count actual tools including zero-tool sessions"});

function purpose() { return {description: "Count actual tool records, including sessions with zero tools"}; }
__verb__("purpose", {name: "purpose", short: "Explain this query"});
