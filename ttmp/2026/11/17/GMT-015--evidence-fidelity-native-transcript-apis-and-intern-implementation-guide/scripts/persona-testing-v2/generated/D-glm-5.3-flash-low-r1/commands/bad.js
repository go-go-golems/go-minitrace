__verb__("bad", {
  name: "bad",
  short: "Sibling command with a negative numeric metadata default (-1)",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Path to a normalized .minitrace.json archive to load",
    },
    threshold: {
      type: "int",
      default: -1,
      help: "Negative numeric default sibling flag",
    },
  },
});

function bad(source, threshold) {
  const mt = require("minitrace");
  const db = mt.db().File(source).SQLiteMemory().Build();
  try {
    const sidRow = db.queryOne("SELECT session_id FROM sessions LIMIT 1");
    if (!sidRow || !sidRow.session_id) {
      throw new Error("no session found in source archive");
    }
    const recipe = mt.query().ToolRows().SessionID(sidRow.session_id).Build();
    const row = db.queryOne(
      "SELECT session_id, COUNT(*) AS tool_count, " + Number(threshold) + " AS threshold FROM (" + recipe.sql() + ") GROUP BY session_id",
      ...recipe.args()
    );
    return [row];
  } finally {
    db.close();
  }
}
