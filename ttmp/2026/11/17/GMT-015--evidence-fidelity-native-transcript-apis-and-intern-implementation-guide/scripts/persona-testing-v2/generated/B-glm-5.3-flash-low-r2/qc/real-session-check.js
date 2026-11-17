function realSessionCheck() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try {
    const out = {};
    const sess = db.query("SELECT session_id, agent_framework FROM sessions");
    out.sessions = sess;
    const counts = db.execute(mt.query().SessionToolCounts().Build());
    if (counts.error) { out.count_error = counts.error; }
    out.tool_counts = counts.rows;
    const tok = db.execute(mt.query().TokenUsageRows().Build());
    if (tok.error) { out.token_error = tok.error; }
    out.token_rows = tok.rows;
    out.token_complete = tok.complete === true;
    return out;
  } finally { db.close(); }
}

__verb__("realSessionCheck", {
  name: "real-session-check",
  short: "Tool counts and token evidence for loaded real archives",
  fields: {}
});
