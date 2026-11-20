// good.js — structured JS query command (scanner-first).
// Single self-named verb "good", so the CLI path collapses to:
//   go-minitrace query commands good
// REQUIRED string flag --source selects a normalized minitrace archive file.
// The handler opens that source, runs the standard ToolRows recipe with a
// bound session filter, and returns exactly one JSON row:
//   { session_id: <id>, tool_count: <n> }

__section__("parameters", {
  title: "Parameters",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Path to a normalized minitrace archive file (.minitrace.json) (required)",
    },
  },
});

function good(parameters) {
  const mt = require("minitrace");

  // Build a read-only in-memory normalized SQLite DB from exactly the
  // requested source archive (no cache writes).
  const sources = mt.sources().Archive(parameters.source).Build();
  const cache = mt.cache().Memory().Build();
  const db = mt.db().Sources(sources).Cache(cache).Build();
  try {
    // The archive is a per-session archive; resolve its session identity
    // from the normalized sessions table using bound SQL.
    const idRows = db.query("SELECT session_id FROM sessions ORDER BY session_id");
    if (idRows.length !== 1) {
      throw new Error(
        "expected exactly one session in source archive " + parameters.source +
        ", found " + idRows.length
      );
    }
    const sessionId = idRows[0].session_id;

    // Standard ToolRows recipe, bound to the resolved session id.
    const recipe = mt.query().ToolRows().SessionID(sessionId).Build();
    const rows = db.query(recipe.sql(), ...recipe.args());

    // Exactly one JSON row with top-level session_id and tool_count.
    return [{ session_id: sessionId, tool_count: rows.length }];
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Emit one {session_id, tool_count} row for a source archive",
  long: "Loads the normalized archive given by --source and counts tool rows for its session using the standard ToolRows recipe with bound SQL.",
  fields: {
    parameters: { bind: "parameters" },
  },
});
