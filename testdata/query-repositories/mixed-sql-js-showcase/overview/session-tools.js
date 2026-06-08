__section__("filters", {
  title: "Filters",
  fields: {
    limit: { type: "int", default: 20, help: "Maximum number of sessions to inspect" },
  },
});

function _db(mt) {
  return mt.db().RuntimeArchives().QueryCommandDefaults().Build();
}

function sessionList(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  return db.query(`
    SELECT session_id AS id, title, agent_framework AS framework
    FROM sessions
    ORDER BY started_at DESC
    LIMIT ${filters.limit}
  `);
}

function frameworkShare(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  const rows = db.query(`
    SELECT agent_framework AS framework, COUNT(*) AS count
    FROM sessions
    GROUP BY 1
    ORDER BY count DESC, framework ASC
    LIMIT ${filters.limit}
  `);
  const total = rows.reduce((sum, row) => sum + Number(row.count || 0), 0) || 1;
  return rows.map(row => ({ ...row, share_percent: Math.round((Number(row.count || 0) / total) * 100) }));
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions from JS",
  fields: { filters: { bind: "filters" } },
});

__verb__("frameworkShare", {
  name: "framework-share",
  short: "Compute framework share percentages in JS after querying",
  fields: { filters: { bind: "filters" } },
});
