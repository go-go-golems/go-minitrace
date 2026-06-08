const transforms = require("./lib/transforms");

__section__("filters", {
  title: "Filters",
  fields: {
    framework: { type: "stringList", help: "Restrict commands to specific agent frameworks" },
    limit: { type: "int", default: 20, help: "Maximum number of sessions to inspect" },
  },
});

function _db(mt) {
  return mt.db().RuntimeArchives().QueryCommandDefaults().Build();
}

function _withDB(mt, fn) {
  const db = _db(mt);
  try {
    return fn(db);
  } finally {
    db.close();
  }
}

function _frameworkFilterSql(mt, filters) {
  return filters.framework?.length
    ? `AND agent_framework IN (${mt.sql.stringIn(filters.framework)})`
    : "";
}

function sessionList(filters) {
  const mt = require("minitrace");
  return _withDB(mt, (db) => db.query(`
    SELECT
      session_id AS id,
      title,
      agent_framework AS framework
    FROM sessions
    WHERE 1=1
    ${_frameworkFilterSql(mt, filters)}
    ORDER BY started_at DESC
    LIMIT ${filters.limit}
  `));
}

function frameworkShare(filters) {
  const mt = require("minitrace");
  return _withDB(mt, (db) => {
    const rows = db.query(`
    SELECT
      agent_framework AS framework,
      COUNT(*) AS count
    FROM sessions
    WHERE 1=1
    ${_frameworkFilterSql(mt, filters)}
    GROUP BY 1
    ORDER BY count DESC, framework ASC
    LIMIT ${filters.limit}
  `);

    return transforms.addSharePercent(rows, "count");
  });
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
