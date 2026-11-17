// bad.js — probe: metadata parameter with numeric default -1.
// Sibling of good.js in the same query repository.

__section__("meta", {
  title: "Meta",
  fields: {
    offset: {
      type: "int",
      default: -1,
      help: "Numeric offset with a negative default (-1 means 'unset')",
    },
  },
});

function bad(meta) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query(
      `SELECT COUNT(*) AS n FROM tool_calls WHERE ? = ?`,
      [meta.offset, meta.offset]
    );
  } finally {
    db.close();
  }
}

__verb__("bad", {
  name: "bad",
  short: "Probe numeric default of -1 on a metadata parameter",
  fields: {
    meta: { bind: "meta" },
  },
});
