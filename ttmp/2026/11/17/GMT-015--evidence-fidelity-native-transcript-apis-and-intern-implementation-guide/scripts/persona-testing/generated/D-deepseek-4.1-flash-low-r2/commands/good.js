// good.js -- root verb "good": per-session tool counts over runtime archives.
// Required provenance flag plus the ordinary runtime archive flags.
__section__("inputs", {
  title: "Inputs",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Required provenance label for the supplied runtime archives",
    },
  },
});

function good(inputs) {
  const mt = require("minitrace");
  const db = mt.db()
    .RuntimeArchives()
    .SQLiteMemory()          // no disk cache writes
    .MaxCellBytes(2000000)   // enough for raw_json/result evidence
    .Build();
  try {
    // Introspection through the API, not sqlite_master.
    const tableNames = db.tables().map((t) => t.name);
    const schemaTableNames = db.schema().tables.map((t) => t.name);

    // Standard recipe, executed with the recipe's own bound args.
    const recipe = mt.query().ToolRows().Build();
    const env = db.execute(recipe);
    if (env.error || !env.complete) {
      throw new Error(env.error || "incomplete ToolRows evidence");
    }

    // Explicit bound parameter (never string interpolation) for the
    // sessions projection.
    const sessions = db.query(
      "SELECT session_id, title, agent_framework FROM sessions WHERE COALESCE(agent_framework,'') <> ? ORDER BY session_id",
      "",
    );

    const bySession = {};
    for (const row of env.rows) {
      const id = row.session_id;
      if (!bySession[id]) {
        bySession[id] = { session_id: id, tool_count: 0, tools: {} };
      }
      bySession[id].tool_count += 1;
      const name = row.tool_name || "(unknown)";
      bySession[id].tools[name] = (bySession[id].tools[name] || 0) + 1;
    }

    return sessions.map((s) => {
      const agg = bySession[s.session_id] || { tool_count: 0, tools: {} };
      return {
        source: inputs.source,
        session_id: s.session_id,
        agent_framework: s.agent_framework,
        title: s.title,
        tool_count: agg.tool_count,
        tool_names: Object.keys(agg.tools).sort().join(","),
        table_names: tableNames.join(","),
        schema_table_names: schemaTableNames.join(","),
      };
    });
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Per-session tool counts for supplied runtime archives",
  fields: { inputs: { bind: "inputs" } },
});
