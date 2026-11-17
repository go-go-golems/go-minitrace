function _withDB(mt, fn) {
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try {
    return fn(db);
  } finally {
    db.close();
  }
}

function sessionCounts() {
  const mt = require("minitrace");
  return _withDB(mt, (db) =>
    db.query(`
      SELECT
        s.session_id,
        s.agent_framework,
        (SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id)      AS turn_count,
        (SELECT COUNT(*) FROM tool_calls c WHERE c.session_id = s.session_id) AS tool_count,
        (SELECT MIN(t.timestamp) FROM turns t WHERE t.session_id = s.session_id) AS first_turn_ts,
        (SELECT MAX(t.timestamp) FROM turns t WHERE t.session_id = s.session_id) AS last_turn_ts
      FROM sessions s
      ORDER BY s.session_id
    `)
  );
}

function topTools() {
  const mt = require("minitrace");
  return _withDB(mt, (db) =>
    db.query(`
      WITH ranked AS (
        SELECT s.session_id,
               COALESCE(NULLIF(c.tool_name,''), '<unnamed>') AS tool_name,
               COUNT(*) AS n,
               ROW_NUMBER() OVER (PARTITION BY s.session_id ORDER BY COUNT(*) DESC,
                 COALESCE(NULLIF(c.tool_name,''),'<unnamed>') ASC) AS rn
        FROM tool_calls c JOIN sessions s ON s.session_id = c.session_id
        GROUP BY 1,2
      )
      SELECT session_id, tool_name, n, rn FROM ranked WHERE rn <= 3
      ORDER BY session_id, rn
    `)
  );
}

__verb__("sessionCounts", {
  name: "session-counts",
  short: "Per-session actual turn and tool row counts (same semantics as SQL)",
  fields: {},
});

__verb__("topTools", {
  name: "top-tools",
  short: "Per-session top-3 tool names by row count",
  fields: {},
});
