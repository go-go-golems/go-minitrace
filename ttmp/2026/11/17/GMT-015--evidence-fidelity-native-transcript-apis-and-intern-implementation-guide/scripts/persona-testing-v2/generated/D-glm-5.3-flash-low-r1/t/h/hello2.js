__verb__("hello2", {
  name: "hello2",
  short: "Hello2",
  fields: {
    source: { type: "string", required: true, help: "src path" },
  },
});
function hello2(source) {
  const mt = require("minitrace");
  const db = mt.db().File(source).SQLiteMemory().Build();
  try { return db.query("SELECT session_id FROM sessions LIMIT 1"); } finally { db.close(); }
}
