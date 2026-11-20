const completeRows = function(result) {
  if (result.error || !result.complete) {
    throw new Error(result.error || "Incomplete query result");
  }
  return result.rows;
};

const {queryFilters} = require("../lib/query-filters");

const transforms = require("./lib/transforms");

__section__("filters", {
  title: "Filters",
  fields: {
    framework: { type: "stringList", help: "Restrict commands to specific agent frameworks" },
    limit: { type: "int", default: 20, help: "Maximum number of sessions to inspect" },
  },
});

function _db(mt) {
  return mt.db().sources(mt.sources().runtimeArchives().build())
    .import({autoConvert: true, strict: true}).build();
}

function _withDB(mt, fn) {
  const db = _db(mt);
  try {
    return fn(db);
  } finally {
    db.close();
  }
}

function sessionList(filters) {
  const mt = require("minitrace");
  const {frameworkKeys} = queryFilters(filters);
  return _withDB(mt, (db) => completeRows(db.execute(mt.sql`
    SELECT
      session_id AS id,
      title,
      agent_framework AS framework
    FROM sessions
    WHERE 1=1
    AND (${frameworkKeys} IS NULL OR (agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(agent_framework) || '|') > 0))
    ORDER BY started_at DESC
    LIMIT ${filters.limit}
  `.build())));
}

function frameworkShare(filters) {
  const mt = require("minitrace");
  const {frameworkKeys} = queryFilters(filters);
  return _withDB(mt, (db) => {
    const rows = completeRows(db.execute(mt.sql`
    SELECT
      agent_framework AS framework,
      COUNT(*) AS count
    FROM sessions
    WHERE 1=1
    AND (${frameworkKeys} IS NULL OR (agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(agent_framework) || '|') > 0))
    GROUP BY 1
    ORDER BY count DESC, framework ASC
    LIMIT ${filters.limit}
  `.build()));

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
