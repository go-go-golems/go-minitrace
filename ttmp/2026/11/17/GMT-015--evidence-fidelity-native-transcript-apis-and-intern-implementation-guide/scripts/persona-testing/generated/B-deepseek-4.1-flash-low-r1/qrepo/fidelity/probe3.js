function tok(f) {
  const mt = require("minitrace");
  const db = mt.db().Files(["/private/tmp/gmt015-persona/inputs/archive.json"]).SQLiteMemory().Build();
  try {
    const r = mt.query().TokenUsageRows().ByTurn().Build();
    const tr = mt.query().ToolRows().Build();
    return [{tokensql: r.sql(), tokenargs: JSON.stringify(r.args()), toolsql: tr.sql()}];
  } finally { db.close(); }
}
__verb__("tok", { name: "tok", short: "tok", fields: {} });
