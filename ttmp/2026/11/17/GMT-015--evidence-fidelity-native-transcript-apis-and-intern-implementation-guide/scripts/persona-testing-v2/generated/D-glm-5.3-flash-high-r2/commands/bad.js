// commands/bad.js — sibling verb "bad" for diagnostics.
// Same shape as good.js, plus one numeric field whose default is the
// negative integer -1 declared in JS __verb__ metadata (not via CLI or
// YAML defaults). Kept present so catalog loading and --help must
// tolerate it.

__section__("source_args", {
  title: "Source",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Path to a normalized minitrace archive file (.minitrace.json)",
    },
  },
});

function bad(args) {
  const mt = require("minitrace");
  const db = mt.db().File(args.source).SQLiteMemory().Build();
  try {
    const sess = db.queryOne(
      "SELECT session_id FROM sessions ORDER BY session_id LIMIT 1"
    );
    if (!sess || !sess.session_id) {
      throw new Error("archive contains no session id: " + args.source);
    }
    const recipe = mt.query().ToolRows().SessionID(sess.session_id).Build();
    const result = db.execute(recipe);
    if (result.error || !result.complete) {
      throw new Error(result.error || "Incomplete evidence");
    }
    return [
      {
        session_id: sess.session_id,
        tool_count: result.rows.length,
        threshold: args.threshold,
      },
    ];
  } finally {
    db.close();
  }
}

__verb__("bad", {
  name: "bad",
  short: "Diagnostic sibling with a negative numeric default in verb metadata",
  fields: {
    source_args: { bind: "source_args" },
    threshold: {
      type: "int",
      default: -1,
      help: "Numeric default declared in __verb__ metadata (negative on purpose)",
    },
  },
});
