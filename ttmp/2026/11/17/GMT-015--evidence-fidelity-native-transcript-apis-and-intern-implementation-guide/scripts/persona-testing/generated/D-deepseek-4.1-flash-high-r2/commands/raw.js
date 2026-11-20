__section__("filters", {
  title: "Filters",
  fields: {
    source: { type: "string", required: true, help: "Source label for raw shape demo" }
  }
});

function raw(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    const rows = db.query("SELECT session_id, tool_call_count FROM sessions WHERE (? = 'all' OR agent_framework = ?) ORDER BY session_id", filters.source, filters.source);
    // Returns ONE object with nested raw JS values instead of an array of row objects.
    return {
      source: filters.source,
      row_count: rows.length,
      raw_rows: rows,
      tool_counts: rows.map((r) => r.tool_call_count),
      nested: { first: rows[0] || null },
      array_is_array: Array.isArray(rows)
    };
  } finally {
    db.close();
  }
}

__verb__("raw", {
  name: "raw",
  short: "Return one row containing raw JS arrays/objects",
  fields: { filters: { bind: "filters" } }
});
