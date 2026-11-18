const completeRows = function(result) {
  if (result.error || !result.complete) {
    throw new Error(result.error || "Incomplete query result");
  }
  return result.rows;
};

const {queryFilters} = require("../lib/query-filters");

__section__("filters", {
  title: "Filters",
  fields: {
    framework: { type: "stringList", help: "Restrict commands to specific agent frameworks" },
    limit: { type: "int", default: 3, help: "Maximum number of rows to return" },
  },
});

function _db(mt) {
  return mt.db().sources(mt.sources().runtimeArchives().build())
    .import({autoConvert: true, strict: true}).build();
}

async function _withDBAsync(mt, fn) {
  const db = _db(mt);
  try {
    return await fn(db);
  } finally {
    db.close();
  }
}

async function delayedSummary(filters) {
  const timer = require("timer");
  const mt = require("minitrace");
  const {frameworkKeys} = queryFilters(filters);
  await timer.sleep(1);
  return _withDBAsync(mt, async (db) => {
    const summary = completeRows(db.execute(mt.sql`
    SELECT
      COUNT(*) AS session_count,
      MIN(session_id) AS first_id,
      MAX(agent_framework) AS sample_framework
    FROM sessions
    WHERE 1=1
    AND (${frameworkKeys} IS NULL OR (agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(agent_framework) || '|') > 0))
  `.build()))[0] ?? null;

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
  const {frameworkKeys} = queryFilters(filters);
  await timer.sleep(1);
  return _withDBAsync(mt, async (db) => {
    const rows = completeRows(db.execute(mt.sql`
    SELECT
      session_id AS id,
      title,
      agent_framework AS framework
    FROM sessions
    WHERE 1=1
    AND (${frameworkKeys} IS NULL OR (agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(agent_framework) || '|') > 0))
    ORDER BY started_at DESC
    LIMIT ${filters.limit}
  `.build()));

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
  short: "Build a summary row with async JS and a tagged SQL plan",
  fields: { filters: { bind: "filters" } },
});

__verb__("topSessionCards", {
  name: "top-session-cards",
  short: "Use async JS to reshape query rows into card-like output",
  fields: { filters: { bind: "filters" } },
});
