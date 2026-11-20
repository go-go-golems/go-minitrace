__section__("inputs", {
  title: "Inputs",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Normalized .minitrace.json archive file path",
    },
  },
});

function good(inputs) {
  const mt = require("minitrace");
  const source = inputs.source;
  if (!source) throw new Error("source is required");

  const manifest = mt.corpusManifest([source]);
  if (manifest.conflictCount || manifest.entries.length !== 1) {
    throw new Error("source must identify exactly one archive");
  }
  const sessionId = manifest.entries[0].sessionId;

  const sources = mt.sources().File(source).Build();
  const db = mt.db().Sources(sources).SQLiteMemory().StrictEvidence(true).Build();
  try {
    const recipe = mt.query().ToolRows().SessionID(sessionId).Build();
    const result = db.execute({
      sql: `SELECT s.session_id AS session_id,
                   COUNT(t.session_id) AS tool_count
            FROM sessions s
            LEFT JOIN (${recipe.sql()}) AS t
              ON t.session_id = s.session_id
            WHERE s.session_id = ?
            GROUP BY s.session_id`,
      args: [...recipe.args(), sessionId],
    });
    if (result.error || !result.complete) {
      throw new Error(result.error || "incomplete evidence");
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
    inputs: { bind: "inputs" },
  },
});
