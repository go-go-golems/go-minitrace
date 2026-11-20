__section__("source", {
  title: "Source",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Normalized archive file path",
    },
  },
});

function good(fields) {
  const mt = require("minitrace");
  const db = mt.db()
    .File(fields.source)
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
  fields: { source: { bind: "source" } },
});
