const completeRows = function(result) {
  if (result.error || !result.complete) {
    throw new Error(result.error || "Incomplete query result");
  }
  return result.rows;
};

const {queryFilters} = require("../lib/query-filters");

const cookbook = require("./lib/cookbook");

__section__("filters", {
  title: "Filters",
  fields: {
    framework: { type: "stringList", help: "Restrict to specific agent frameworks" },
    working_directory_like: { type: "string", help: "Substring match for working directory" },
    limit: { type: "int", default: 8, help: "Maximum number of rows to return" }
  }
});

function _db(mt) { return mt.db().sources(mt.sources().runtimeArchives().build())
    .import({autoConvert: true, strict: true}).build(); }

function _withDB(mt, fn) {
  const db = _db(mt);
  try {
    return fn(db);
  } finally {
    db.close();
  }
}

function toolboxOverview(filters) {
  const mt = require("minitrace");
  const {frameworkKeys, cwd} = queryFilters(filters);
  return _withDB(mt, (db) => {

  const toolRows = completeRows(db.execute(mt.sql`
    SELECT t.tool_name, COUNT(*) AS tool_uses, COUNT(DISTINCT t.session_id) AS session_count
    FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
    WHERE (${frameworkKeys} IS NULL OR (s.agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(s.agent_framework) || '|') > 0))
    AND (${cwd} IS NULL OR COALESCE(s.working_directory, '') LIKE ${cwd}) AND t.tool_name IS NOT NULL
    GROUP BY 1
    ORDER BY tool_uses DESC, tool_name ASC
    LIMIT ${filters.limit}
  `.build()));

  const operationRows = completeRows(db.execute(mt.sql`
    SELECT t.tool_name, t.operation_type, COUNT(*) AS use_count
    FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
    WHERE (${frameworkKeys} IS NULL OR (s.agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(s.agent_framework) || '|') > 0))
    AND (${cwd} IS NULL OR COALESCE(s.working_directory, '') LIKE ${cwd}) AND t.tool_name IS NOT NULL
    GROUP BY 1, 2
    ORDER BY tool_name ASC, use_count DESC, operation_type ASC
  `.build()));

  const workspaceRows = completeRows(db.execute(mt.sql`
    SELECT t.tool_name, COALESCE(s.working_directory, '(none)') AS working_directory, COUNT(*) AS tool_uses
    FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
    WHERE (${frameworkKeys} IS NULL OR (s.agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(s.agent_framework) || '|') > 0))
    AND (${cwd} IS NULL OR COALESCE(s.working_directory, '') LIKE ${cwd}) AND t.tool_name IS NOT NULL
    GROUP BY 1, 2
    ORDER BY tool_name ASC, tool_uses DESC, working_directory ASC
  `.build()));

  const topOperationByTool = cookbook.firstBy(operationRows, "tool_name");
  const topWorkspaceByTool = cookbook.firstBy(workspaceRows, "tool_name");

  return toolRows.map((row, index) => {
    const op = topOperationByTool[row.tool_name] || {};
    const workspace = topWorkspaceByTool[row.tool_name] || {};
    return {
      rank: index + 1,
      tool_name: row.tool_name,
      tool_uses: row.tool_uses,
      session_count: row.session_count,
      reuse_density: cookbook.round2(cookbook.safeNumber(row.tool_uses) / Math.max(cookbook.safeNumber(row.session_count), 1)),
      dominant_operation: op.operation_type || "",
      dominant_workspace: workspace.working_directory || "",
      dominant_workspace_slug: cookbook.shortWorkspace(workspace.working_directory),
    };
  });
  });
}

function toolPairMatrix(filters) {
  const mt = require("minitrace");
  const {frameworkKeys, cwd} = queryFilters(filters);
  return _withDB(mt, (db) => {
  const sessionToolRows = completeRows(db.execute(mt.sql`
    SELECT DISTINCT t.session_id AS id, t.tool_name
    FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
    WHERE (${frameworkKeys} IS NULL OR (s.agent_framework IS NOT NULL AND instr(${frameworkKeys}, '|' || hex(s.agent_framework) || '|') > 0))
    AND (${cwd} IS NULL OR COALESCE(s.working_directory, '') LIKE ${cwd}) AND t.tool_name IS NOT NULL
    ORDER BY id ASC, tool_name ASC
  `.build()));

  return cookbook.pairCounts(sessionToolRows)
    .slice(0, filters.limit)
    .map((row, index) => ({ rank: index + 1, ...row }));
  });
}

__verb__("toolboxOverview", {
  name: "toolbox-overview",
  short: "Join multiple tool aggregates into a tool intelligence table",
  fields: { filters: { bind: "filters" } }
});

__verb__("toolPairMatrix", {
  name: "tool-pair-matrix",
  short: "Compute tool co-occurrence pairs in JavaScript from session-tool rows",
  fields: { filters: { bind: "filters" } }
});
