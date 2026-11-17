// tool-rows.js — exercise the standard mt.query().ToolRows() recipe.
// The recipe itself supplies SQL and bound arguments; execute() returns the
// fidelity envelope, so we inspect complete/error instead of treating an
// empty array as proof of zero tools.

function toolRows() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    const recipe = mt.query().ToolRows().Build();
    const result = db.execute(recipe);
    if (result.error && !result.complete) {
      return {
        error: result.error,
        complete: result.complete,
        count: result.count || 0,
        rows: [],
      };
    }
    return {
      recipe_name: recipe.name(),
      recipe_sql: recipe.sql(),
      recipe_args: recipe.args(),
      count: result.count,
      complete: result.complete,
      truncated: !!result.truncated,
      // Keep the artifact bounded while still proving the recipe ran end to end.
      row_preview: (result.rows || []).slice(0, 5),
      total_rows_returned: (result.rows || []).length,
    };
  } finally {
    db.close();
  }
}

__verb__("toolRows", {
  name: "tool-rows",
  short: "Execute the standard ToolRows recipe and return its fidelity envelope",
});
