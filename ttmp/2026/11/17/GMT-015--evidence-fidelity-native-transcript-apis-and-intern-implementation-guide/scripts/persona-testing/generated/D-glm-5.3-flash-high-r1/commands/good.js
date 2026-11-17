// good.js — root verb "good": per-session tool counts for supplied runtime archives.
// Repository layout: commands/good.js (self-named single-verb file => collapsed CLI path
// "query commands good" when loaded with --query-repository <this workspace root>).
//
// Exercises:
//   - required string flag "source" (agent framework filter)
//   - runtime archives via mt.db().RuntimeArchives() (--archive-glob on the CLI)
//   - bound SQL parameters (positional "?" placeholders)
//   - the standard ToolRows recipe via mt.query().ToolRows().Build()
//   - allowed schema introspection via db.tables() / db.schema()
//   - one intentional disallowed SQL introspection query (sqlite_master), captured as a
//     diagnostic instead of aborting the command.

__section__("filters", {
  title: "Filters",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Agent framework to report (claude-code, codex, pi)",
    },
    min_tools: {
      type: "int",
      default: 1,
      help: "Minimum tool count for a session to be included",
    },
  },
});

function good(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try {
    // Allowed introspection APIs (documented alternative to sqlite_master).
    const tables = db.tables();
    const schemaRaw = db.schema();
    const schemaTables = schemaRaw && Array.isArray(schemaRaw.tables) ? schemaRaw.tables : [];

    // Per-session tool counts with BOUND SQL parameters (positional ? placeholders).
    const counts = db.query(
      `SELECT s.session_id,
              s.agent_framework,
              COUNT(t.tool_call_id) AS tool_count
         FROM sessions s
         LEFT JOIN tool_calls t ON t.session_id = s.session_id
        WHERE s.agent_framework = ?
          AND t.tool_call_id IS NOT NULL
        GROUP BY s.session_id, s.agent_framework
       HAVING COUNT(t.tool_call_id) >= ?
        ORDER BY tool_count DESC, s.session_id ASC`,
      filters.source,
      filters.min_tools
    );

    // Standard ToolRows recipe, executed with the full evidence envelope.
    const toolRecipe = mt.query().ToolRows().Build();
    const toolResult = db.execute(toolRecipe);
    if (toolResult.error) {
      throw new Error("ToolRows recipe failed: " + toolResult.error);
    }

    // Intentional disallowed SQL introspection query, expected to be rejected by the
    // read-only sandbox. Captured as a diagnostic; does not abort the command.
    let introspection = null;
    try {
      db.query("SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name");
      introspection = { allowed: true };
    } catch (e) {
      introspection = { allowed: false, error: String(e && e.message ? e.message : e) };
    }

    // Array return: CLI renders one table row per element.
    return counts.map((row) => ({
      source: row.agent_framework,
      session_id: row.session_id,
      tool_count: row.tool_count,
      normalized_tool_rows: toolResult.count,
      tables_introspected: tables.length,
      schema_entries: schemaTables.length,
      sqlite_master_rejected: !introspection.allowed,
      sqlite_master_error: introspection.error || "",
    }));
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Per-session tool counts for the supplied runtime archives",
  fields: {
    filters: { bind: "filters" },
  },
});
