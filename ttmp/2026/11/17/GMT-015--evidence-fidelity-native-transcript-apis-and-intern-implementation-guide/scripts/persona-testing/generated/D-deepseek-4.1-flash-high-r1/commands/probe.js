__section__("inputs", {
  title: "Inputs",
  fields: {
    source: { type: "string", required: true, help: "Agent framework/source label" },
  },
});

function probe(inputs) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try {
    const tables = db.tables();
    const schema = db.schema();
    const bad = db.queryResult("SELECT name FROM sqlite_master WHERE type='table'");
    return [{
      source_type: typeof inputs.source,
      source_value: inputs.source,
      tables_type: Array.isArray(tables) ? "array" : typeof tables,
      tables: tables,
      schema_type: Array.isArray(schema) ? "array" : typeof schema,
      schema: schema,
      disallowed_error: bad.error || null,
      disallowed_count: bad.count,
    }];
  } finally {
    db.close();
  }
}

__verb__("probe", {
  name: "probe",
  short: "Probe db.tables()/db.schema() and disallowed sqlite_master access",
  fields: { inputs: { bind: "inputs" } },
});
