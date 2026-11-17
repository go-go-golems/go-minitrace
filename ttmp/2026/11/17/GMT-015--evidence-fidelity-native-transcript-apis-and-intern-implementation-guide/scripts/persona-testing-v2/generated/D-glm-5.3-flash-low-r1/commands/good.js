__verb__("good", {
  name: "good",
  short: "Return one row with session_id and tool_count from a required source archive",
  fields: {
    source: {
      type: "string",
      required: true,
      help: "Path to a normalized .minitrace.json archive to load",
    },
  },
});

function good(source) {
  const mt = require("minitrace");
  const db = mt.db().File(source).SQLiteMemory().Build();
  try {
    const sidRow = db.queryOne("SELECT session_id FROM sessions LIMIT 1");
    if (!sidRow || !sidRow.session_id) {
      throw new Error("no session found in source archive");
    }
    const recipe = mt.query().ToolRows().SessionID(sidRow.session_id).Build();
    const row = db.queryOne(
      "SELECT session_id, COUNT(*) AS tool_count FROM (" + recipe.sql() + ") GROUP BY session_id",
      ...recipe.args()
    );
    if (!row) throw new Error("no tool rows for session " + sidRow.session_id);
    return [row];
  } finally {
    db.close();
  }
}
