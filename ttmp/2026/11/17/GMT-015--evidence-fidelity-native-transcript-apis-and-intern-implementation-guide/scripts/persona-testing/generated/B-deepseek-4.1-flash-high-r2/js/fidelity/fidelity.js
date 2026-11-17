// Tool-evidence fidelity: source_ordinal / arguments_json / output_status vs
// outcome_status, archive-array order vs chronological order, and the two
// recipe styles (builder recipe vs bound-argument SQL) including how
// query/queryOne throw while queryResult/execute return envelopes.

__section__("opts", {
  title: "Options",
  fields: {
    session_id: { type: "string", default: "", help: "Optional session filter" },
    tool_id: { type: "string", default: "a-second", help: "Bound-argument tool id" },
  },
});

function _db(mt) {
  return mt.db().RuntimeArchives().SQLiteMemory().MaxCellBytes(4096).StrictEvidence(false).Build();
}

function _thrown(fn) {
  try {
    return { threw: false, value: fn() };
  } catch (e) {
    return { threw: true, error: String(e && e.message ? e.message : e) };
  }
}

// Builder recipe path: mt.query().ToolRows() -> recipe.sql()/recipe.args() ->
// db.execute(recipe).
function builderRecipe(opts) {
  const mt = require("minitrace");
  const db = _db(mt);
  try {
    const recipe = mt.query().ToolRows().Build();
    const env = db.execute(recipe);
    return {
      probe: "builder-recipe",
      recipeName: recipe.name(),
      recipeSql: recipe.sql(),
      recipeArgs: recipe.args(),
      complete: env.complete,
      count: env.count,
      error: env.error || null,
      columns: env.columns,
      rows: env.rows,
    };
  } finally {
    db.close();
  }
}

// Bound-argument SQL path: same projection shape, but explicit columns and ?
// placeholders instead of Go-built args.
function boundArgs(opts) {
  const mt = require("minitrace");
  const db = _db(mt);
  try {
    const sql =
      "SELECT tool_call_id, source_ordinal, emitting_turn_index AS turn_index, " +
      "timestamp, tool_name, outcome_status, output_status, success, arguments_json " +
      "FROM tool_calls WHERE tool_call_id = ? OR tool_name = ? " +
      "ORDER BY source_ordinal";
    const env = db.queryResult(sql, opts.tool_id, "synthetic_read");
    return {
      probe: "bound-args",
      sql: sql,
      args: [opts.tool_id, "synthetic_read"],
      complete: env.complete,
      count: env.count,
      rows: env.rows,
      // query() is rows-only, so it cannot report completeness at all
      queryRows: db.query(sql, opts.tool_id, "synthetic_read"),
      queryOne: db.queryOne("SELECT COUNT(*) AS n FROM tool_calls"),
    };
  } finally {
    db.close();
  }
}

// Exception contrast: query/queryOne throw, queryResult/execute return
// envelopes carrying `error`.
function errorSurfaces(opts) {
  const mt = require("minitrace");
  const db = _db(mt);
  try {
    const badSyntax = "SELEC 1";
    const badTable = "SELECT * FROM no_such_table";
    return {
      probe: "error-surfaces",
      query_badSyntax: _thrown(() => db.query(badSyntax)),
      queryOne_badSyntax: _thrown(() => db.queryOne(badSyntax)),
      query_badTable: _thrown(() => db.query(badTable)),
      queryResult_badSyntax: db.queryResult(badSyntax),
      execute_badSyntax: db.execute({ sql: badSyntax, args: [] }),
      queryResult_badTable: db.queryResult(badTable),
      queryResult_noRows: db.queryResult("SELECT tool_call_id FROM tool_calls WHERE 1=0"),
      queryOne_noRows: _thrown(() => db.queryOne("SELECT tool_call_id FROM tool_calls WHERE 1=0")),
      queryResult_forbiddenTable: db.queryResult("SELECT name FROM sqlite_master"),
    };
  } finally {
    db.close();
  }
}

// Archive-array order vs chronological order for the same three tool records.
function toolOrder(opts) {
  const mt = require("minitrace");
  const db = _db(mt);
  try {
    const projection =
      "SELECT tool_call_id, source_ordinal, timestamp, tool_name, record_kind, " +
      "outcome_status, output_status, success, arguments_json " +
      "FROM tool_calls WHERE session_id = ?";
    const archiveOrder = db.query(projection + " ORDER BY source_ordinal", "persona-synthetic");
    const chronological = db.query(projection + " ORDER BY timestamp, tool_call_id", "persona-synthetic");
    // NOTE: db.query(<recipe>) throws "only SELECT and WITH queries are allowed"
    // because query() expects SQL text; recipes must go through execute() or
    // recipe.sql()/recipe.args(). Capture that here as documented friction.
    const recipe = mt.query().ToolRows().Build();
    const recipeDirectQuery = _thrown(() => db.query(recipe));
    const recipeDefault = db.query(recipe.sql(), ...recipe.args());
    const recipeExecute = db.execute(recipe);
    return {
      probe: "tool-order",
      archiveOrder: archiveOrder,
      chronological: chronological,
      recipeDirectQueryError: recipeDirectQuery.threw ? recipeDirectQuery.error : "did not throw",
      recipeExecuteOrder: (recipeExecute.rows || []).map((r) => r.tool_call_id),
      recipeDefaultOrder: (recipeDefault || []).map((r) => ({
        tool_call_id: r.tool_call_id,
        source_ordinal: r.source_ordinal,
        timestamp: r.timestamp,
      })),
      sourceOrdinals: (archiveOrder || []).map((r) => r.source_ordinal),
      archiveIds: (archiveOrder || []).map((r) => r.tool_call_id),
      chronologicalIds: (chronological || []).map((r) => r.tool_call_id),
    };
  } finally {
    db.close();
  }
}

__verb__("builderRecipe", {
  name: "builder-recipe",
  short: "Builder recipe via execute()",
  fields: { opts: { bind: "opts" } },
});
__verb__("boundArgs", {
  name: "bound-args",
  short: "Bound-argument SQL query",
  fields: { opts: { bind: "opts" } },
});
__verb__("errorSurfaces", {
  name: "error-surfaces",
  short: "Compare thrown exceptions with error envelopes",
  fields: { opts: { bind: "opts" } },
});
__verb__("toolOrder", {
  name: "tool-order",
  short: "Archive order vs chronological order",
  fields: { opts: { bind: "opts" } },
});
