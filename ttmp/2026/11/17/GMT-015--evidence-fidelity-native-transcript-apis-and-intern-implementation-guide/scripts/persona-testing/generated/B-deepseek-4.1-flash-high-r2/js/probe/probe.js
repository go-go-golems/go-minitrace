__section__("opts", {
  title: "Options",
  fields: {
    label: { type: "string", default: "probe", help: "label" },
  },
});

function probe(opts) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().MaxCellBytes(1000).StrictEvidence(false).Build();
  try {
    const tables = db.tables();
    const recipe = mt.query().ToolRows().Build();
    return {
      label: opts.label,
      tables: tables,
      recipeName: recipe.name(),
      recipeSQL: recipe.sql(),
      recipeArgs: recipe.args(),
      runtime: mt.runtime,
      utf8: mt.utf8ByteLength("abé"),
      sources: db.sources(),
    };
  } finally {
    db.close();
  }
}

__verb__("probe", {
  name: "probe",
  short: "Probe runtime surface",
  fields: { opts: { bind: "opts" } },
});
