// bad.js — negative int default probe.
//
// This file declares a metadata parameter with a signed numeric default
// (default: -1). It is kept in place intentionally to test whether the
// scanner/loader supports signed defaults and whether a bad sibling file
// breaks the valid good.js command in the same repository.

__section__("inputs", {
  title: "Inputs",
  fields: {
    min_tools: {
      type: "int",
      default: -1,
      help: "Negative-default probe: minimum tool count filter (-1 = disabled)",
    },
  },
});

function bad(inputs) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query("SELECT COUNT(*) AS session_count FROM sessions");
  } finally {
    db.close();
  }
}

__verb__("bad", {
  name: "bad",
  short: "Probe negative int default handling in scanner metadata",
  fields: { inputs: { bind: "inputs" } },
});
