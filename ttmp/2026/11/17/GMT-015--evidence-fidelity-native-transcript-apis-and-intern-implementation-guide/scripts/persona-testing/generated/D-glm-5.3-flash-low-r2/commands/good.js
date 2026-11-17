// good.js — self-named single-verb JS query command: per-session tool counts.
// CLI path (collapsed): go-minitrace query commands good \
//   --query-repository ./commands --archive-glob ...

__section__("sourceopts", {
  title: "Source",
  fields: {
    source: { type: "string", help: "Agent framework filter bound into SQL (e.g. claude-code)", required: true },
    limit: { type: "int", default: 5, help: "Max rows" },
  },
});

function good(sourceopts) {
  const mt = require("minitrace");
  if (!sourceopts.source) throw new Error("--source is required");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    // Schema introspection (documented API): tables() and schema()
    const tables = db.tables().map(function (t) { return t.name || t.table || JSON.stringify(t).slice(0, 40); });
    const schemaJson = JSON.stringify(db.schema()).slice(0, 300);

    // Bound SQL parameters via query(sql, ...args)
    const sessions = db.query(
      "SELECT session_id, agent_framework AS framework, title, tool_call_count " +
      "FROM sessions WHERE agent_framework = ? ORDER BY session_id LIMIT ?",
      sourceopts.source, sourceopts.limit
    );

    // Standard ToolRows recipe, executed with the full evidence envelope.
    const recipe = mt.query().ToolRows().SessionID(sessions[0].session_id).Build();
    const recipeResult = db.execute(recipe);
    const recipeNote = recipeResult.error
      ? "error: " + String(recipeResult.error).slice(0, 160)
      : (recipeResult.complete ? "complete rowCount=" + recipeResult.count : "incomplete rowCount=" + recipeResult.count + " truncated=" + recipeResult.truncated + " rowsTruncated=" + recipeResult.rowsTruncated);

    // Per-session tool counts, bound parameters
    const counts = db.query(
      "SELECT tc.session_id, s.agent_framework AS framework, COUNT(*) AS tool_calls " +
      "FROM tool_calls tc JOIN sessions s ON s.session_id = tc.session_id " +
      "WHERE s.agent_framework = ? GROUP BY 1, 2 ORDER BY 3 DESC LIMIT ?",
      sourceopts.source, sourceopts.limit
    );

    // Intentionally disallowed SQL introspection query (sandbox forbids
    // sqlite_master). We capture the diagnostic instead of crashing.
    const introspection = db.queryResult("SELECT name FROM sqlite_master WHERE type='table'");
    const introspectionError = introspection.error || (introspection.complete ? null : "incomplete");

    // Contrast: array of row objects (CLI table formatting) plus raw JS values
    // (nested object preserved only via --output json).
    return counts.map(function (r) { return r; }).concat([
      {
        session_id: "RECIPE_TOOLROWS " + recipeNote + " session=" + sessions[0].session_id,
        framework: "tables=" + tables.join(","),
        title: "schema=" + schemaJson,
        tool_calls: introspectionError ? "introspection blocked: " + String(introspectionError).slice(0, 160) : "allowed?!"
      },
    ]);
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Per-session tool counts over runtime archives",
  fields: { sourceopts: { bind: "sourceopts" } },
});
