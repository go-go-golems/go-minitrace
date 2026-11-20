__section__("filters", {
  title: "Filters",
  fields: {
    source: { type: "string", required: true, help: "Source label for the probe" }
  }
});

function probe(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    // Deliberately disallowed SQLite introspection. Note the apostrophe in sqlite_master.
    const envelope = db.queryResult(
      "SELECT name, type FROM sqlite_master WHERE type IN ('table','view') ORDER BY name"
    );
    return {
      source: filters.source,
      disallowed_sql_error: envelope.error || "",
      returned_rows: envelope.count,
      allowed_table_count: db.tables().length,
      allowed_table_names: db.tables().map((t) => t.name)
    };
  } finally {
    db.close();
  }
}

__verb__("probe", {
  name: "probe",
  short: "Probe a disallowed sqlite_master introspection query",
  fields: { filters: { bind: "filters" } }
});
