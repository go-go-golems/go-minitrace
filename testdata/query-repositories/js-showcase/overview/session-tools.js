const transforms = require("./lib/transforms");

__section__("filters", {
  title: "Filters",
  fields: {
    framework: {
      type: "stringList",
      help: "Restrict commands to specific agent frameworks",
    },
    limit: {
      type: "int",
      default: 20,
      help: "Maximum number of sessions to inspect",
    },
  },
});

function _frameworkFilterSql(mt, filters) {
  // Parenthesize DuckDB JSON-arrow predicates. The -> / ->> operators have low
  // precedence, so the wrapped form is easier to read and less brittle.
  return filters.framework?.length
    ? `AND (environment->>'agent_framework') IN (${mt.sql.stringIn(filters.framework)})`
    : "";
}

function sessionList(filters) {
  const mt = require("minitrace");
  return mt.legacy.query(`
    SELECT
      id,
      title,
      environment->>'agent_framework' AS framework
    FROM ${mt.legacy.tableName}
    WHERE 1=1
    ${_frameworkFilterSql(mt, filters)}
    ORDER BY timing->>'started_at' DESC
    LIMIT ${filters.limit}
  `);
}

function frameworkShare(filters) {
  const mt = require("minitrace");
  const rows = mt.legacy.query(`
    SELECT
      environment->>'agent_framework' AS framework,
      COUNT(*) AS count
    FROM ${mt.legacy.tableName}
    WHERE 1=1
    ${_frameworkFilterSql(mt, filters)}
    GROUP BY 1
    ORDER BY count DESC, framework ASC
    LIMIT ${filters.limit}
  `);

  return transforms.addSharePercent(rows, "count");
}

__verb__("sessionList", {
  name: "session-list",
  short: "List sessions from JS",
  fields: {
    filters: { bind: "filters" },
  },
});

__verb__("frameworkShare", {
  name: "framework-share",
  short: "Compute framework share percentages in JS after querying",
  fields: {
    filters: { bind: "filters" },
  },
});
