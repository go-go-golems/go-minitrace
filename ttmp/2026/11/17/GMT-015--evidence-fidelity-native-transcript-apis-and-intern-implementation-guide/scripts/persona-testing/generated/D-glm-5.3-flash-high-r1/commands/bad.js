// bad.js — intentionally invalid sibling: a metadata parameter whose numeric
// default is the SIGNED value -1. Kept in place for grading.
//
// Hypothesis under test: the scanner-first metadata extractor only accepts
// plain positive numeric literals; "-1" is a tree-sitter unary_expression,
// not a "number" node, so scanning this file is expected to fail.
__section__("filters", {
  title: "Filters",
  fields: {
    min_tools: {
      type: "int",
      default: -1,
      help: "Minimum tool count (negative sentinel = include everything)",
    },
  },
});

function bad(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try {
    return db.query(
      `SELECT s.session_id, COUNT(t.tool_call_id) AS tool_count
         FROM sessions s
         LEFT JOIN tool_calls t ON t.session_id = s.session_id
        WHERE t.tool_call_id IS NOT NULL
        GROUP BY s.session_id
       HAVING COUNT(t.tool_call_id) >= ?
        ORDER BY tool_count DESC`,
      filters.min_tools
    );
  } finally {
    db.close();
  }
}

__verb__("bad", {
  name: "bad",
  short: "INVALID: metadata parameter with signed numeric default -1",
  fields: {
    filters: { bind: "filters" },
  },
});
