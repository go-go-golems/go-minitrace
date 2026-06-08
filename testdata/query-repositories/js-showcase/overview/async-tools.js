__section__("filters", {
  title: "Filters",
  fields: {
    framework: { type: "stringList", help: "Restrict commands to specific agent frameworks" },
    limit: { type: "int", default: 3, help: "Maximum number of rows to return" },
  },
});

function _db(mt) {
  return mt.db().RuntimeArchives().QueryCommandDefaults().Build();
}

async function _withDBAsync(mt, fn) {
  const db = _db(mt);
  try {
    return await fn(db);
  } finally {
    db.close();
  }
}

function _frameworkFilterSql(mt, filters) {
  return filters.framework?.length
    ? `AND agent_framework IN (${mt.sql.stringIn(filters.framework)})`
    : "";
}

async function delayedSummary(filters) {
  const timer = require("timer");
  const mt = require("minitrace");
  await timer.sleep(1);
  return _withDBAsync(mt, async (db) => {
    const summary = db.queryOne(`
    SELECT
      COUNT(*) AS session_count,
      MIN(session_id) AS first_id,
      MAX(agent_framework) AS sample_framework
    FROM sessions
    WHERE 1=1
    ${_frameworkFilterSql(mt, filters)}
  `);

    return {
      delayed: true,
      requested_limit: filters.limit,
      session_count: summary?.session_count || 0,
      first_id: summary?.first_id || "",
      sample_framework: summary?.sample_framework || "",
    };
  });
}

async function topSessionCards(filters) {
  const timer = require("timer");
  const mt = require("minitrace");
  await timer.sleep(1);
  return _withDBAsync(mt, async (db) => {
    const rows = db.query(`
    SELECT
      session_id AS id,
      title,
      agent_framework AS framework
    FROM sessions
    WHERE 1=1
    ${_frameworkFilterSql(mt, filters)}
    ORDER BY started_at DESC
    LIMIT ${filters.limit}
  `);

    return rows.map((row, index) => ({
      rank: index + 1,
      id: row.id,
      framework: row.framework,
      title: row.title,
      card_label: `#${index + 1} ${row.title}`,
    }));
  });
}

__verb__("delayedSummary", {
  name: "delayed-summary",
  short: "Use async JS plus queryOne to build a summary row",
  fields: { filters: { bind: "filters" } },
});

__verb__("topSessionCards", {
  name: "top-session-cards",
  short: "Use async JS to reshape query rows into card-like output",
  fields: { filters: { bind: "filters" } },
});
