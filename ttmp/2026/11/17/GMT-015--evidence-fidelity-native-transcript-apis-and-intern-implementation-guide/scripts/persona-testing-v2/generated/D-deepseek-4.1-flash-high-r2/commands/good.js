__section__("filters", {
  title: "Source",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Path to one normalized .minitrace.json archive",
    },
  },
});

function good(filters) {
  const mt = require("minitrace");
  const sources = mt.sources().File(filters.source).Build();
  const db = mt.db()
    .Sources(sources)
    .SQLiteMemory()
    .StrictEvidence(true)
    .Build();
  try {
    const session = db.queryOne(
      "SELECT session_id FROM sessions ORDER BY session_id LIMIT 1"
    );
    if (!session) {
      throw new Error("no normalized session loaded from source");
    }

    // Standard ToolRows recipe, used as the tool-evidence ledger.
    const recipe = mt.query().ToolRows().SessionID(session.session_id).Build();
    const result = db.execute({
      sql: `SELECT s.session_id AS session_id,
                   COUNT(tr.tool_call_id) AS tool_count
            FROM sessions s
            LEFT JOIN (${recipe.sql()}) tr
              ON tr.session_id = s.session_id
            WHERE s.session_id = ?
            GROUP BY s.session_id`,
      args: [...recipe.args(), session.session_id],
    });
    if (result.error || !result.complete) {
      throw new Error(result.error || "incomplete tool-row evidence");
    }
    if (result.rows.length !== 1) {
      throw new Error("expected exactly one result row, got " + result.rows.length);
    }
    return result.rows;
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Return session_id and tool_count for one normalized archive",
  fields: {
    filters: { bind: "filters" },
  },
});
