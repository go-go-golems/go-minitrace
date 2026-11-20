const mt = require("minitrace");
const fixture = {id: "fixture", profile: "organic", turns: [], tool_calls: []};
const db = mt.db().sources(mt.sources().content(JSON.stringify(fixture)).build()).build();
try {
  const tools = mt.sql`SELECT 'a' AS session_id, 1 AS id UNION ALL SELECT 'a', 2 UNION ALL SELECT 'b', 3`
    .columns({session_id: mt.type.string(), id: mt.type.integer()});
  const counts = tools.select(t => ({
    id: t.id,
    peers: tools.where(u => u.session_id.eq(t.session_id)).count(),
  })).orderBy(row => row.id.asc());
  const result = db.execute(counts.build());
  if (!result.complete || JSON.stringify(result.rows.map(r => r.peers)) !== "[2,2,1]") {
    throw Error(JSON.stringify(result));
  }
} finally {
  db.close();
}
