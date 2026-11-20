function audit() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  return db.query("SELECT session_id FROM sessions LIMIT 1");
}
__verb__("audit", {name: "audit", short: "Inspect complete normalized tool rows"});
