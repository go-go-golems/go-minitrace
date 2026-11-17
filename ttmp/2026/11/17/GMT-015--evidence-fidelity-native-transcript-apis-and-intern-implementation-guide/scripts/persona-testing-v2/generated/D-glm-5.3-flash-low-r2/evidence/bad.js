// bad.js — sibling command whose metadata carries a numeric default of -1.
//
// NOTE (measured): the JS metadata scanner rejects negative numeric literals.
// Both `default: -1` in __section__ field metadata and `metadata: {x: -1}` in
// __verb__ metadata fail with:
//   unsupported metadata literal "unary_expression"
// causing the file to be skipped at catalog load with a diagnostic on stderr.
// The -1 literal is kept here on purpose so the bad-sibling diagnostic is
// observable whenever the catalog is loaded.
__section__("filters", {
  title: "Filters",
  fields: {
    threshold: {
      type: "int",
      default: -1,
      help: "Threshold value with a negative numeric default",
    },
  },
});

function bad(filters) {
  const mt = require("minitrace");
  const db = mt.db()
    .RuntimeArchives()
    .SQLiteMemory()
    .Build();
  try {
    const result = db.execute(mt.query().ToolRows().Build());
    if (result.error || !result.complete) {
      throw new Error(result.error || "Incomplete evidence");
    }
    return result.rows;
  } finally {
    db.close();
  }
}

__verb__("bad", {
  name: "bad",
  short: "Sibling command carrying a negative numeric default (-1) in metadata",
  fields: {
    filters: { bind: "filters" },
  },
  metadata: {
    numeric_default: -1,
  },
});
