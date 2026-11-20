// bad.js — sibling probe: a metadata parameter with a numeric default of -1.
// Purpose: test whether the scanner/compiler supports signed numeric defaults.

function bad(params) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query(
      "SELECT session_id, agent_framework AS framework, tool_call_count FROM sessions " +
      "WHERE tool_call_count >= ? ORDER BY tool_call_count DESC LIMIT ?",
      params.min_tool_calls, params.limit
    );
  } finally {
    db.close();
  }
}

__verb__("bad", {
  name: "bad",
  short: "Probe: int parameter with signed default -1",
  fields: {
    min_tool_calls: { type: "int", default: -1, help: "Minimum tool_call_count (signed default probe)" },
    limit: { type: "int", default: 5, help: "Max rows" },
  },
});
