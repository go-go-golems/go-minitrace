// good.js — one-session tool count from a single normalized archive.
__section__("filters", {
  title: "Filters",
  fields: {
    source: {
      type: "string",
      help: "Path to a normalized minitrace archive file",
      required: true,
    },
  },
});

function good(filters) {
  const mt = require("minitrace");
  if (!filters.source) {
    throw new Error("source is required: pass a normalized archive file path");
  }
  const db = mt.db()
    .Sources(mt.sources().File(filters.source).Build())
    .SQLiteMemory()
    .MaxCellBytes(1048576)
    .Build();
  try {
    const result = db.execute(mt.query().ToolRows().Build());
    if (result.error || !result.complete) {
      throw new Error(result.error || "Incomplete evidence");
    }
    const counts = new Map();
    for (const row of result.rows) {
      const sid = row.session_id;
      counts.set(sid, (counts.get(sid) || 0) + 1);
    }
    if (counts.size !== 1) {
      throw new Error("expected exactly one session in source archive, got " + counts.size);
    }
    const [sessionId, toolCount] = [...counts.entries()][0];
    return [{ session_id: sessionId, tool_count: toolCount }];
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Return session_id and tool_count for one archive",
  fields: {
    filters: { bind: "filters" },
  },
});
