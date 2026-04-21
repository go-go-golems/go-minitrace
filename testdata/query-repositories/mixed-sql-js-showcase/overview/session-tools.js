const transforms = require("./lib/transforms");

__section__("filters", {
  title: "Filters",
  fields: {
    framework: { type: "stringList", help: "Restrict to specific agent frameworks" },
    limit: { type: "int", default: 10, help: "Maximum number of rows to return" }
  }
});

function _frameworkFilterSql(mt, filters) {
  return filters.framework?.length
    ? `AND environment->>'agent_framework' IN (${mt.sql.stringIn(filters.framework)})`
    : "";
}

function sessionList(filters) {
  const mt = require("minitrace");
  return mt.query(`
    SELECT id, title, environment->>'agent_framework' AS framework
    FROM ${mt.tableName}
    WHERE 1=1
    ${_frameworkFilterSql(mt, filters)}
    ORDER BY timing->>'started_at' DESC
    LIMIT ${filters.limit}
  `);
}

function frameworkShare(filters) {
  const mt = require("minitrace");
  const rows = mt.query(`
    SELECT environment->>'agent_framework' AS framework, COUNT(*) AS count
    FROM ${mt.tableName}
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
  fields: { filters: { bind: "filters" } }
});

__verb__("frameworkShare", {
  name: "framework-share",
  short: "Compute framework shares in JS",
  fields: { filters: { bind: "filters" } }
});
