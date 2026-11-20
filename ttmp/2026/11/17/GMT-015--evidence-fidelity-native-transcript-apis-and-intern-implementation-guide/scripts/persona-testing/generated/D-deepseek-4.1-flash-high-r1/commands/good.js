__section__("inputs", {
  title: "Inputs",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Required. Agent framework/source to report (claude-code, codex, or pi)",
    },
  },
});

function good(inputs) {
  const mt = require("minitrace");
  const source = String(inputs.source || "").trim();
  if (!source) throw new Error("--source is required");

  const db = mt.db().RuntimeArchives().SQLiteMemory().MaxCellBytes(1048576).StrictEvidence(true).Build();
  try {
    // Schema introspection must go through the JS API, not SQL system tables.
    const tables = db.tables();
    const schema = db.schema();
    const schemaTableCount = (schema && schema.tables) ? schema.tables.length : 0;

    // Bound SQL parameter: source selects the agent framework.
    const sessions = db.query(
      "SELECT session_id, agent_framework FROM sessions WHERE agent_framework = ? ORDER BY session_id",
      source
    );

    // Standard ToolRows recipe; aggregate one row per tool record per session.
    const recipe = mt.query().ToolRows().Build();
    const result = db.execute(recipe);
    if (result.error) throw new Error("ToolRows failed: " + result.error);
    if (!result.complete) throw new Error("ToolRows returned incomplete evidence");

    const counts = {};
    for (const row of result.rows) {
      const sid = row.session_id;
      counts[sid] = (counts[sid] || 0) + 1;
    }

    // Intentional disallowed introspection: must fail closed with a diagnostic.
    const disallowed = db.queryResult("SELECT name FROM sqlite_master WHERE type='table'");

    return sessions.map((s) => ({
      source: s.agent_framework,
      session_id: s.session_id,
      tool_count: counts[s.session_id] || 0,
      runtime_table_count: Array.isArray(tables) ? tables.length : 0,
      schema_table_count: schemaTableCount,
      disallowed_introspection_error: disallowed.error || null,
    }));
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Per-session tool counts for supplied runtime archives",
  fields: { inputs: { bind: "inputs" } },
});
