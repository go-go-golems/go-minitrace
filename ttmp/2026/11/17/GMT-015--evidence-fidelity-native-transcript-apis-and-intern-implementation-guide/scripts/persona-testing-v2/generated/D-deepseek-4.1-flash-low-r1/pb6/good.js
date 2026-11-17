function good(source) {
  const mt = require("minitrace");
  const db = mt.db()
    .File(source)
    .SQLiteMemory()
    .StrictEvidence(true)
    .Build();
  try {
    const recipe = mt.query().ToolRows().Build();
    const result = db.execute({
      sql: `SELECT session_id, COUNT(*) AS tool_count
            FROM (${recipe.sql()})
            GROUP BY session_id`,
      args: recipe.args(),
    });
    if (result.error || !result.complete) {
      throw new Error(result.error || "Incomplete evidence");
    }
    return result.rows;
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Count tool rows for one normalized archive",
  params: [
    {
      name: "source",
      type: "string",
      required: true,
      bind: "source",
      help: "Normalized archive file path",
    },
  ],
});
