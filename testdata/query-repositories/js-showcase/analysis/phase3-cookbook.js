__section__("filters", {
  title: "Filters",
  fields: {
    working_directory_like: { type: "string", help: "Substring match for working directory" },
    framework: { type: "stringList", help: "Restrict to specific agent frameworks" },
    limit: { type: "int", default: 10, help: "Maximum number of rows to return" }
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

function _sessionWhere(mt, filters, alias) {
  const p = alias ? `${alias}.` : "";
  const clauses = ["1=1"];
  if (filters.framework?.length) clauses.push(`${p}agent_framework IN (${mt.sql.stringIn(filters.framework)})`);
  if (filters.working_directory_like) clauses.push(`COALESCE(${p}working_directory, '') LIKE ${mt.sql.like(filters.working_directory_like)}`);
  return clauses.join("\n    AND ");
}

function _limit(filters) {
  const limit = Number(filters.limit || 10);
  return Math.max(1, Math.min(limit, 100));
}

function contextInventory(filters) {
  const mt = require("minitrace");
  const whereSql = _sessionWhere(mt, filters, "s");
  return _withDB(mt, (db) => {
  const rows = db.query(`
    SELECT
      s.session_id AS id,
      s.title,
      COALESCE(s.working_directory, '(none)') AS working_directory,
      s.git_branch,
      s.autonomy_level,
      s.sandbox,
      s.human_attention,
      s.duration_seconds,
      m.total_input_tokens,
      m.total_output_tokens,
      m.total_cache_read_tokens,
      m.total_tool_tokens,
      COALESCE(a.annotation_count, 0) AS annotation_count,
      COALESCE(h.handover_count, 0) AS handover_count,
      COALESCE(sa.spawned_agent_count, 0) AS spawned_agent_count
    FROM sessions s
    LEFT JOIN metrics m ON m.session_id = s.session_id
    LEFT JOIN (
      SELECT session_id, COUNT(*) AS annotation_count
      FROM annotations
      GROUP BY session_id
    ) a ON a.session_id = s.session_id
    LEFT JOIN (
      SELECT session_id, COUNT(*) AS handover_count
      FROM handovers
      GROUP BY session_id
    ) h ON h.session_id = s.session_id
    LEFT JOIN (
      SELECT session_id, COUNT(*) AS spawned_agent_count
      FROM tool_calls
      WHERE spawned_agent_type IS NOT NULL
      GROUP BY session_id
    ) sa ON sa.session_id = s.session_id
    WHERE ${whereSql}
    ORDER BY annotation_count DESC, spawned_agent_count DESC, s.tool_call_count DESC, id ASC
    LIMIT ${_limit(filters)}
  `);

  return rows.map((row) => ({
    ...row,
    sandbox_label: row.sandbox === 1 || row.sandbox === true ? "sandboxed" : "unspecified",
    context_label: [row.git_branch || "no-branch", row.autonomy_level || "no-autonomy", row.human_attention || "no-attention"].join(" / ")
  }));
  });
}

function annotationRiskMatrix(filters) {
  const mt = require("minitrace");
  const whereSql = _sessionWhere(mt, filters, "s");
  return _withDB(mt, (db) => {
  return db.query(`
    SELECT
      a.category,
      COALESCE(a.classification, '(none)') AS classification,
      a.scope_type,
      COUNT(*) AS annotation_count,
      COUNT(DISTINCT a.session_id) AS session_count,
      GROUP_CONCAT(DISTINCT s.title) AS sample_titles
    FROM annotations a
    JOIN sessions s ON s.session_id = a.session_id
    WHERE ${whereSql}
    GROUP BY a.category, a.classification, a.scope_type
    ORDER BY annotation_count DESC, category ASC, classification ASC
    LIMIT ${_limit(filters)}
  `);
  });
}

function handoverQueue(filters) {
  const mt = require("minitrace");
  const whereSql = _sessionWhere(mt, filters, "s");
  return _withDB(mt, (db) => {
  return db.query(`
    SELECT
      h.direction,
      h.from_session,
      h.to_session,
      s.session_id AS id,
      s.title,
      h.state_description,
      SUBSTR(h.document, 1, 160) AS document_preview
    FROM handovers h
    JOIN sessions s ON s.session_id = h.session_id
    WHERE ${whereSql}
    ORDER BY h.direction ASC, s.started_at DESC, id ASC
    LIMIT ${_limit(filters)}
  `);
  });
}

function spawnedAgentAudit(filters) {
  const mt = require("minitrace");
  const whereSql = _sessionWhere(mt, filters, "s");
  return _withDB(mt, (db) => {
  return db.query(`
    SELECT
      s.session_id AS id,
      s.title,
      t.tool_call_id,
      t.tool_name,
      t.operation_type,
      t.spawned_agent_type,
      t.spawned_agent_task_scope,
      t.spawned_agent_sub_session_id,
      t.spawned_agent_outcome_summary,
      t.position_in_session
    FROM tool_calls t
    JOIN sessions s ON s.session_id = t.session_id
    WHERE ${whereSql}
      AND t.spawned_agent_type IS NOT NULL
    ORDER BY s.started_at DESC, t.emitting_turn_index ASC, t.tool_call_id ASC
    LIMIT ${_limit(filters)}
  `);
  });
}

__verb__("contextInventory", {
  name: "context-inventory",
  short: "Inventory Phase 3 context, token, annotation, handover, and spawned-agent coverage",
  fields: { filters: { bind: "filters" } }
});

__verb__("annotationRiskMatrix", {
  name: "annotation-risk-matrix",
  short: "Group annotations by category, classification, and scope over normalized annotation rows",
  fields: { filters: { bind: "filters" } }
});

__verb__("handoverQueue", {
  name: "handover-queue",
  short: "List received and produced handover documents attached to sessions",
  fields: { filters: { bind: "filters" } }
});

__verb__("spawnedAgentAudit", {
  name: "spawned-agent-audit",
  short: "List tool calls that spawned subagents and their outcomes",
  fields: { filters: { bind: "filters" } }
});
