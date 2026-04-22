__section__("filters", {
  title: "Filters",
  fields: {
    framework: {
      type: "stringList",
      help: "Restrict commands to specific agent frameworks",
    },
    limit: {
      type: "int",
      default: 3,
      help: "Maximum number of rows to return",
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

async function delayedSummary(filters) {
  const timer = require("timer");
  const mt = require("minitrace");
  await timer.sleep(1);

  const summary = mt.queryOne(`
    SELECT
      COUNT(*) AS session_count,
      MIN(id) AS first_id,
      MAX(environment->>'agent_framework') AS sample_framework
    FROM ${mt.tableName}
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
}

async function topSessionCards(filters) {
  const timer = require("timer");
  const mt = require("minitrace");
  await timer.sleep(1);

  const rows = mt.query(`
    SELECT
      id,
      title,
      environment->>'agent_framework' AS framework
    FROM ${mt.tableName}
    WHERE 1=1
    ${_frameworkFilterSql(mt, filters)}
    ORDER BY timing->>'started_at' DESC
    LIMIT ${filters.limit}
  `);

  return rows.map((row, index) => ({
    rank: index + 1,
    id: row.id,
    framework: row.framework,
    title: row.title,
    card_label: `#${index + 1} ${row.title}`,
  }));
}

__verb__("delayedSummary", {
  name: "delayed-summary",
  short: "Use async JS plus queryOne to build a summary row",
  fields: {
    filters: { bind: "filters" },
  },
});

__verb__("topSessionCards", {
  name: "top-session-cards",
  short: "Use async JS to reshape query rows into card-like output",
  fields: {
    filters: { bind: "filters" },
  },
});
