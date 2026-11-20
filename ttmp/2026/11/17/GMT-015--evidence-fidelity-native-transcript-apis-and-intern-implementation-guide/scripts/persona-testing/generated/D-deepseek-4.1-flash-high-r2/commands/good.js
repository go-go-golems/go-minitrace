__section__("filters", {
  title: "Filters",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Required source label: an agent_framework value (claude-code, codex, pi) or 'all'"
    }
  }
});

function good(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().MaxRows(5000).MaxCellBytes(1000000).Build();
  try {
    // Bound parameters: the required flag is passed as a SQLite argument, never interpolated.
    const sessions = db.query(
      `SELECT session_id, agent_framework, source_format, title, turn_count, tool_call_count
         FROM sessions
        WHERE (? = 'all' OR agent_framework = ?)
        ORDER BY session_id`,
      filters.source, filters.source
    );

    // Standard named recipe. execute() returns the full envelope, including completeness.
    const recipe = mt.query().ToolRows().Build();
    const envelope = db.execute(recipe);
    if (envelope.error) throw new Error("ToolRows envelope error: " + envelope.error);

    // Raw JS array (not emitted directly): aggregate it in JS before returning CLI rows.
    const rawToolRows = envelope.rows;
    const bySession = {};
    for (const tool of rawToolRows) {
      if (!bySession[tool.session_id]) bySession[tool.session_id] = [];
      bySession[tool.session_id].push(tool);
    }

    // db.schema() and db.tables() introspection (not sqlite_master).
    const schema = db.schema();
    const tables = db.tables();

    return sessions.map((s) => {
      const tools = bySession[s.session_id] || [];
      const names = [...new Set(tools.map((t) => t.tool_name).filter((n) => n !== null))].sort();
      return {
        source: filters.source,
        session_id: s.session_id,
        agent_framework: s.agent_framework,
        source_format: s.source_format,
        title: s.title,
        turn_count: s.turn_count,
        session_tool_call_count: s.tool_call_count,
        tool_rows_from_recipe: tools.length,
        tool_name_count: names.length,
        tool_names: names,
        sample_raw_tool_values: tools.slice(0, 2).map((t) => ({
          tool_name: t.tool_name,
          turn_index: t.turn_index,
          success: t.success,
          outcome_status: t.outcome_status
        })),
        recipe_name: recipe.name(),
        recipe_arg_count: recipe.args().length,
        schema_version: schema.version,
        dialect: schema.dialect,
        table_count: tables.length,
        table_names: tables.map((t) => t.name),
        raw_recipe_rows_are_array: Array.isArray(rawToolRows),
        recipe_complete: envelope.complete,
        recipe_truncated: !!envelope.truncated,
        recipe_rows_truncated: !!envelope.rowsTruncated,
        recipe_cell_diagnostic_count: (envelope.cellDiagnostics || []).length
      };
    });
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Report per-session tool counts for runtime archives",
  fields: { filters: { bind: "filters" } }
});
