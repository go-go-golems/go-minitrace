const cookbook = require("./lib/cookbook");

__section__("filters", {
  title: "Filters",
  fields: {
    framework: { type: "stringList", help: "Restrict to specific agent frameworks" },
    working_directory_like: { type: "string", help: "Substring match for working directory" },
    limit: { type: "int", default: 8, help: "Maximum number of rows to return" }
  }
});

function _db(mt) { return mt.db().RuntimeArchives().QueryCommandDefaults().Build(); }

function _withDB(mt, fn) {
  const db = _db(mt);
  try {
    return fn(db);
  } finally {
    db.close();
  }
}

function _toolWhere(mt, filters) {
  const clauses = ["1=1"];
  if (filters.framework?.length) clauses.push(`s.agent_framework IN (${mt.sql.stringIn(filters.framework)})`);
  if (filters.working_directory_like) clauses.push(`COALESCE(s.working_directory, '') LIKE ${mt.sql.like(filters.working_directory_like)}`);
  return clauses.join("\n    AND ");
}

function toolboxOverview(filters) {
  const mt = require("minitrace");
  const whereSql = _toolWhere(mt, filters);
  return _withDB(mt, (db) => {

  const toolRows = db.query(`
    SELECT t.tool_name, COUNT(*) AS tool_uses, COUNT(DISTINCT t.session_id) AS session_count
    FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
    WHERE ${whereSql} AND t.tool_name IS NOT NULL
    GROUP BY 1
    ORDER BY tool_uses DESC, tool_name ASC
    LIMIT ${filters.limit}
  `);

  const operationRows = db.query(`
    SELECT t.tool_name, t.operation_type, COUNT(*) AS use_count
    FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
    WHERE ${whereSql} AND t.tool_name IS NOT NULL
    GROUP BY 1, 2
    ORDER BY tool_name ASC, use_count DESC, operation_type ASC
  `);

  const workspaceRows = db.query(`
    SELECT t.tool_name, COALESCE(s.working_directory, '(none)') AS working_directory, COUNT(*) AS tool_uses
    FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
    WHERE ${whereSql} AND t.tool_name IS NOT NULL
    GROUP BY 1, 2
    ORDER BY tool_name ASC, tool_uses DESC, working_directory ASC
  `);

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
  const whereSql = _toolWhere(mt, filters);
  return _withDB(mt, (db) => {
  const sessionToolRows = db.query(`
    SELECT DISTINCT t.session_id AS id, t.tool_name
    FROM tool_calls t JOIN sessions s ON s.session_id = t.session_id
    WHERE ${whereSql} AND t.tool_name IS NOT NULL
    ORDER BY id ASC, tool_name ASC
  `);

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
