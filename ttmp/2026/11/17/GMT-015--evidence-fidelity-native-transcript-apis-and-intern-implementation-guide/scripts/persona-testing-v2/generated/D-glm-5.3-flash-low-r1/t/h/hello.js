__verb__("hello", {
  name: "hello",
  short: "Hello js",
});
function hello() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    return db.query("SELECT session_id FROM sessions LIMIT 1");
  } finally { db.close(); }
}
