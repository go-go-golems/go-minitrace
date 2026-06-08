__section__("filters", {
  title: "Filters",
  fields: {
    working_directory_like: { type: "string", help: "Substring match for working directory" },
    framework: { type: "stringList", help: "Restrict to specific agent frameworks" },
    limit: { type: "int", default: 20, help: "Maximum number of rows to return" }
  }
});

function _db(mt) { return mt.db().RuntimeArchives().QueryCommandDefaults().Build(); }

function _limit(filters) {
  const limit = Number(filters.limit || 20);
  return Math.max(1, Math.min(limit, 200));
}

function _sessionWhere(mt, filters, alias) {
  const p = alias ? `${alias}.` : "";
  const clauses = ["1=1"];
  if (filters.framework?.length) clauses.push(`${p}agent_framework IN (${mt.sql.stringIn(filters.framework)})`);
  if (filters.working_directory_like) clauses.push(`COALESCE(${p}working_directory, '') LIKE ${mt.sql.like(filters.working_directory_like)}`);
  return clauses.join("\n    AND ");
}

function sessionInventory(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  const whereSql = _sessionWhere(mt, filters, "s");
  return db.query(`
    SELECT
      s.session_id AS id,
      s.title,
      s.agent_framework AS framework,
      COALESCE(s.working_directory, '(none)') AS working_directory,
      s.model,
      s.started_at,
      s.turn_count,
      s.tool_call_count,
      m.total_input_tokens,
      m.total_output_tokens,
      m.session_cost,
      COALESCE(a.annotation_count, 0) AS annotation_count,
      COALESCE(h.handover_count, 0) AS handover_count
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
    WHERE ${whereSql}
    ORDER BY s.started_at DESC, s.tool_call_count DESC, id ASC
    LIMIT ${_limit(filters)}
  `);
}

function toolRiskMatrix(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  const whereSql = _sessionWhere(mt, filters, "s");
  return db.query(`
    SELECT
      t.tool_name,
      t.operation_type,
      COUNT(*) AS calls,
      COUNT(DISTINCT t.session_id) AS sessions,
      SUM(CASE WHEN t.success = 0 THEN 1 ELSE 0 END) AS failed_calls,
      SUM(CASE WHEN t.spawned_agent_type IS NOT NULL THEN 1 ELSE 0 END) AS spawned_agent_calls,
      SUM(CASE WHEN a.annotation_id IS NOT NULL THEN 1 ELSE 0 END) AS annotated_calls,
      ROUND(AVG(COALESCE(t.duration_ms, 0)), 2) AS avg_duration_ms,
      MAX(COALESCE(t.full_bytes, 0)) AS max_full_bytes,
      CASE
        WHEN SUM(CASE WHEN t.success = 0 THEN 1 ELSE 0 END) > 0 THEN 'failure-risk'
        WHEN SUM(CASE WHEN a.annotation_id IS NOT NULL THEN 1 ELSE 0 END) > 0 THEN 'annotated-risk'
        WHEN SUM(CASE WHEN t.spawned_agent_type IS NOT NULL THEN 1 ELSE 0 END) > 0 THEN 'delegation-risk'
        ELSE 'routine'
      END AS risk_label
    FROM tool_calls t
    JOIN sessions s ON s.session_id = t.session_id
    LEFT JOIN annotations a ON a.session_id = t.session_id
      AND a.scope_type = 'tool_call'
      AND a.target_id = t.tool_call_id
    WHERE ${whereSql}
      AND t.tool_name IS NOT NULL
    GROUP BY t.tool_name, t.operation_type
    ORDER BY failed_calls DESC, annotated_calls DESC, spawned_agent_calls DESC, calls DESC, t.tool_name ASC
    LIMIT ${_limit(filters)}
  `);
}

function fileHeatmap(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  const whereSql = _sessionWhere(mt, filters, "s");
  return db.query(`
    SELECT
      f.path,
      f.operation_type,
      COUNT(*) AS touches,
      COUNT(DISTINCT f.session_id) AS sessions,
      SUM(CASE WHEN f.success = 0 THEN 1 ELSE 0 END) AS failed_touches,
      GROUP_CONCAT(DISTINCT s.title) AS sample_titles
    FROM files f
    JOIN sessions s ON s.session_id = f.session_id
    WHERE ${whereSql}
    GROUP BY f.path, f.operation_type
    ORDER BY touches DESC, sessions DESC, f.path ASC
    LIMIT ${_limit(filters)}
  `);
}

function promptInstructionAudit(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  const whereSql = _sessionWhere(mt, filters, "s");
  return db.query(`
    SELECT
      s.session_id AS id,
      s.title,
      s.agent_framework AS framework,
      s.model,
      LENGTH(COALESCE(s.system_prompt, '')) AS system_prompt_chars,
      CASE WHEN COALESCE(s.system_prompt, '') = '' THEN 0 ELSE 1 END AS has_system_prompt,
      CASE WHEN LOWER(COALESCE(s.system_prompt, '')) LIKE '%test%' THEN 1 ELSE 0 END AS mentions_tests,
      CASE WHEN LOWER(COALESCE(s.system_prompt, '')) LIKE '%commit%' THEN 1 ELSE 0 END AS mentions_commits,
      CASE WHEN LOWER(COALESCE(s.system_prompt, '')) LIKE '%legacy%' THEN 1 ELSE 0 END AS mentions_legacy,
      SUBSTR(COALESCE(s.system_prompt, ''), 1, 180) AS prompt_preview
    FROM sessions s
    WHERE ${whereSql}
    ORDER BY has_system_prompt DESC, system_prompt_chars DESC, id ASC
    LIMIT ${_limit(filters)}
  `);
}

function turnTimeline(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  const whereSql = _sessionWhere(mt, filters, "s");
  return db.query(`
    SELECT
      e.session_id AS id,
      s.title AS session_title,
      e.turn_index,
      e.ordinal,
      e.kind,
      e.role,
      e.title,
      e.summary,
      e.severity,
      e.tool_call_id,
      e.annotation_id
    FROM events e
    JOIN sessions s ON s.session_id = e.session_id
    WHERE ${whereSql}
    ORDER BY e.session_id ASC, COALESCE(e.turn_index, 999999) ASC, e.ordinal ASC, e.event_id ASC
    LIMIT ${_limit(filters)}
  `);
}

__verb__("sessionInventory", {
  name: "session-inventory",
  short: "Report session inventory with core metrics, annotation counts, and handover counts",
  fields: { filters: { bind: "filters" } }
});

__verb__("toolRiskMatrix", {
  name: "tool-risk-matrix",
  short: "Report tool risk using failures, annotations, spawned agents, durations, and payload size",
  fields: { filters: { bind: "filters" } }
});

__verb__("fileHeatmap", {
  name: "file-heatmap",
  short: "Report files touched by tools grouped by path and operation type",
  fields: { filters: { bind: "filters" } }
});

__verb__("promptInstructionAudit", {
  name: "prompt-instruction-audit",
  short: "Audit system-prompt instruction coverage with simple SQL heuristics",
  fields: { filters: { bind: "filters" } }
});

__verb__("turnTimeline", {
  name: "turn-timeline",
  short: "Render a normalized turn/tool/annotation event timeline",
  fields: { filters: { bind: "filters" } }
});
