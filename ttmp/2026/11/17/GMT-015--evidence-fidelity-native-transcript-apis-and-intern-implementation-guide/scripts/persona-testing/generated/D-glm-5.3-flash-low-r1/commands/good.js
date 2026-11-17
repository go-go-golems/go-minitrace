// good.js — per-session tool counts over runtime archives.
// Self-named single-verb JS file: verb "good" matches file stem "good",
// so the CLI path collapses to: go-minitrace query commands good

__section__("inputs", {
  title: "Inputs",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Required framework source filter, e.g. claude-code, codex, pi",
    },
  },
});

function good(inputs) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    // 1) Bound SQL parameters (positional args, not string interpolation).
    const counts = db.query(
      `SELECT s.session_id, s.agent_framework, COUNT(*) AS tool_count
       FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
       WHERE s.agent_framework = ?
       GROUP BY s.session_id, s.agent_framework
       ORDER BY s.session_id`,
      [inputs.source]
    );

    // 2) Standard ToolRows recipe, filtered with a bound parameter.
    const recipe = mt.query().ToolRows().Build();
    const toolRowsEnvelope = db.execute({
      sql: `SELECT COUNT(*) AS n FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id WHERE s.agent_framework = ?`,
      args: [inputs.source],
    });
    const recipeInfo = {
      recipe_sql_head: String(recipe.sql()).slice(0, 120),
      recipe_args: recipe.args(),
      toolrows_bound_count: toolRowsEnvelope.rows,
    };

    // 3) Introspection via the supported API (db.tables / db.schema).
    const tables = db.tables();
    const schema = db.schema();
    const schemaInfo = {
      schema_version: schema && schema.version,
      dialect: schema && schema.dialect,
      tables: (schema && schema.tables
        ? schema.tables.map((t) => t.name)
        : []),
      tool_calls_columns: (schema && schema.tables
        ? (schema.tables.find((t) => t.name === "tool_calls") || { columns: [] }).columns.map((c) => c.name)
        : []),
    };

    // 4) Intentional disallowed introspection query: expect a diagnostic.
    let disallowed = null;
    try {
      db.query("SELECT name FROM sqlite_master WHERE type='table'");
      disallowed = { allowed: true };
    } catch (e) {
      disallowed = { allowed: false, error: String(e) };
    }

    return {
      source: inputs.source,
      sessions: counts,
      schema: schemaInfo,
      toolrows_recipe: recipeInfo,
      disallowed_sqlite_master: disallowed,
    };
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Per-session tool counts for supplied runtime archives",
  fields: {
    inputs: { bind: "inputs" },
  },
});
