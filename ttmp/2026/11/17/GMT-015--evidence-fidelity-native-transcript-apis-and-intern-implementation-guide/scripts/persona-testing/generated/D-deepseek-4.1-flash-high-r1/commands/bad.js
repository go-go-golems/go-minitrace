// INTENTIONAL INVALID FIXTURE: the metadata default is a signed number (-1).
// The jsverbs metadata scanner currently rejects negative numeric defaults.
__section__("inputs", {
  title: "Bad inputs",
  fields: {
    limit: {
      type: "int",
      default: -1,
      help: "Signed default that the scanner rejects (expected failure)",
    },
  },
});

function bad(inputs) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try {
    return db.query("SELECT COUNT(*) AS sessions FROM sessions LIMIT ?", inputs.limit);
  } finally {
    db.close();
  }
}

__verb__("bad", {
  name: "bad",
  short: "Invalid fixture: signed numeric metadata default",
  fields: { inputs: { bind: "inputs" } },
});
