// commands/good.js — single self-named verb "good".
// Required string flag --source: path to a normalized minitrace archive.
// Opens that archive and returns exactly one JSON row with top-level
// session_id and tool_count, using the standard ToolRows recipe with a
// bound SessionID (bound SQL args, no string interpolation).

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

function good(args) {
  const mt = require("minitrace");
  const db = mt.db().File(args.source).SQLiteMemory()
    .MaxRows(100000).MaxColumns(256).MaxCellBytes(4000000)
    .Build();
  try {
    const sess = db.queryOne(
      "SELECT session_id FROM sessions ORDER BY session_id LIMIT 1"
    );
    if (!sess || !sess.session_id) {
      throw new Error("archive contains no session id: " + args.source);
    }
    // Standard ToolRows recipe with the session filter bound as a SQL arg.
    const recipe = mt.query().ToolRows().SessionID(sess.session_id).Build();
    const result = db.execute(recipe);
    if (result.error || !result.complete) {
      throw new Error(result.error || "Incomplete evidence");
    }
    return [
      {
        session_id: sess.session_id,
        tool_count: result.rows.length,
      },
    ];
  } finally {
    db.close();
  }
}

__verb__("good", {
  name: "good",
  short: "Return one row: session_id and tool_count for a single archive",
  fields: {
    args: { bind: "source_args" },
  },
});
