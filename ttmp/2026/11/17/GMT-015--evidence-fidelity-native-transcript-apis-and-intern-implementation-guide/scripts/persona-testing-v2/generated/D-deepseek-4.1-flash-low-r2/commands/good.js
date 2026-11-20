__section__("source", {
  title: "Source",
  fields: {
    source: {
      type: "string",
      help: "Path to a normalized .minitrace.json archive",
      required: true,
    },
  },
});

function good(source) {
  const mt = require("minitrace");
  const db = mt.db().File(source.source).SQLiteMemory().MaxCellBytes(1000000).Build();
  try {
    const recipe = mt.query().ToolRows().Build();
    const result = db.execute(recipe);
    if (result.error || !result.complete) {
      throw new Error(result.error || "Incomplete evidence");
    }
    const sessionId = db.queryOne("SELECT session_id FROM sessions LIMIT 1").session_id;
    const count = db.queryOne("SELECT COUNT(*) AS tool_count FROM tool_calls WHERE session_id = ?", sessionId).tool_count;
    return [{ session_id: sessionId, tool_count: count }];
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Return one tool-row count from a required normalized archive source",
  fields: { source: { bind: "source" } },
});
