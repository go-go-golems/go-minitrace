// sqleton
// name: session-counts
// short: Count turns and tool_calls per session via the JS runtime
// flags:
//   archive-glob:
//     type: stringList
//     help: Archive glob for minitrace session files
//     default: []
function sessionCounts() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    const result = db.execute({
      sql: "SELECT s.session_id AS session_id, (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS turn_count, (SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.session_id) AS tool_count FROM sessions s ORDER BY s.session_id"
    });
    if (result.error || !result.complete) {
      throw new Error(result.error || "incomplete evidence");
    }
    return result.rows;
  } finally {
    db.close();
  }
}

__verb__("sessionCounts", {
  name: "session-counts",
  short: "Count turns and tool_calls per session via the JS runtime"
});
