// good.js — per-session tool counts for supplied runtime archives.
//
// Repository-backed JS structured command. Load with:
//   go-minitrace query commands good \
//     --query-repository <this-dir> \
//     --archive-glob <archive.minitrace.json> ... \
//     --source <label>
//
// Demonstrates: required string flag, bound SQL parameters (?),
// standard ToolRows recipe, db.tables()/db.schema() introspection,
// and an intentional disallowed sqlite_master introspection probe
// (captured as a diagnostic, never used as evidence).

__section__("inputs", {
  title: "Inputs",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Required provenance label recorded on every output row",
    },
  },
});

function good(inputs) {
  const mt = require("minitrace");
  // Corpus arguments_json reaches ~60 KB; the 4 KB default cell budget marks
  // the ToolRows envelope incomplete, so raise the per-cell byte budget.
  const db = mt.db()
    .RuntimeArchives()
    .QueryCommandDefaults()
    .MaxCellBytes(262144)
    .Build();
  try {
    // Allowed schema introspection.
    const tableNames = db.tables().map((t) => t.name).sort();
    const schema = db.schema();

    // Intentional disallowed introspection: sqlite_master must be rejected.
    const probe = db.queryResult("SELECT name FROM sqlite_master ORDER BY name");
    const introspectionDiagnostic = probe && probe.error ? probe.error : null;

    // Standard ToolRows recipe: per-tool rows with fidelity columns.
    const recipe = mt.query().ToolRows().Build();
    const toolEnv = db.execute(recipe);
    if (toolEnv.error) {
      throw new Error("ToolRows recipe failed: " + toolEnv.error);
    }
    if (!toolEnv.complete) {
      throw new Error("ToolRows result incomplete; refusing to count");
    }

    // Count tool rows per session purely from the recipe.
    const recipeCounts = {};
    for (const row of toolEnv.rows) {
      const id = row.session_id;
      recipeCounts[id] = (recipeCounts[id] || 0) + 1;
    }

    // Bound-parameter SQL: the required `source` label and each session id
    // are passed as positional arguments, never interpolated into the SQL.
    const sessionIds = Object.keys(recipeCounts).sort();
    const placeholders = sessionIds.map(() => "?").join(", ");
    const args = [String(inputs.source), ...sessionIds];
    const sqlRows = db.query(
      `SELECT
        ? AS source,
        s.session_id AS session_id,
        s.agent_framework AS framework,
        s.source_format AS source_format,
        COUNT(*) AS sql_tool_count
      FROM sessions s
      LEFT JOIN tool_calls tc ON tc.session_id = s.session_id
      WHERE s.session_id IN (${placeholders})
      GROUP BY s.session_id
      ORDER BY s.session_id`,
      ...args
    );

    return sqlRows.map((r) => ({
      source: r.source,
      session_id: r.session_id,
      framework: r.framework,
      source_format: r.source_format,
      sql_tool_count: r.sql_tool_count,
      recipe_tool_count: recipeCounts[r.session_id] !== undefined ? recipeCounts[r.session_id] : 0,
      counts_agree: Number(r.sql_tool_count) === (recipeCounts[r.session_id] !== undefined ? recipeCounts[r.session_id] : 0),
      tables_introspected: tableNames.length,
      schema_version: schema && schema.version ? schema.version : null,
      sqlite_master_diagnostic: introspectionDiagnostic,
    }));
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Report per-session tool counts for supplied runtime archives",
  fields: { inputs: { bind: "inputs" } },
});
