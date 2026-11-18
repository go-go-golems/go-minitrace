// Ordinary CommonJS helpers for transcript analysis. Each helper constructs a
// local query. Call db.execute(query.build()) and inspect its result envelope.
const mt = require("minitrace");

exports.sessionSummary = function(sessionId = null) {
  return mt.sql`
    SELECT
      session_id,
      title,
      summary,
      agent_framework,
      model,
      working_directory,
      started_at,
      ended_at,
      turn_count,
      tool_call_count,
      tool_call_record_count,
      orchestration_count,
      execution_record_count,
      file_change_count,
      model_invocation_count,
      file_touch_count,
      confirmed_file_target_count
    FROM sessions
    WHERE (${sessionId} IS NULL OR session_id = ${sessionId})
    ORDER BY started_at DESC, session_id LIMIT 1
  `;
};

exports.turnRows = function(sessionId = null) {
  return mt.sql`
    SELECT
      session_id,
      turn_index,
      timestamp,
      role,
      source,
      model,
      content_type,
      content,
      thinking,
      input_tokens,
      output_tokens,
      cache_read_tokens,
      cache_creation_tokens,
      reasoning_tokens,
      tool_tokens,
      raw_json
    FROM turns
    WHERE (${sessionId} IS NULL OR session_id = ${sessionId})
    ORDER BY turn_index
  `;
};

exports.toolRows = function(sessionId = null) {
  return mt.sql`
    SELECT
      session_id,
      tool_call_id,
      source_ordinal,
      emitting_turn_index AS turn_index,
      timestamp,
      tool_name,
      operation_type,
      record_kind,
      file_path,
      json_extract(raw_json, '$.input.file_targets') AS file_targets_json,
      command,
      justification,
      arguments_json,
      success,
      outcome_status,
      output_status,
      result,
      error,
      exit_code,
      duration_ms,
      truncated,
      full_bytes,
      full_hash,
      full_reference,
      framework_metadata_json,
      raw_json
    FROM tool_calls
    WHERE (${sessionId} IS NULL OR session_id = ${sessionId})
    ORDER BY COALESCE(emitting_turn_index, 999999), timestamp, tool_call_id
  `.columns({
    session_id: mt.type.string(), tool_call_id: mt.type.string(),
    source_ordinal: mt.type.integer().nullable(), turn_index: mt.type.integer().nullable(),
    timestamp: mt.type.string().nullable(), tool_name: mt.type.string(),
    operation_type: mt.type.string().nullable(), record_kind: mt.type.string(),
    file_path: mt.type.string().nullable(), file_targets_json: mt.type.string().nullable(),
    command: mt.type.string().nullable(), justification: mt.type.string().nullable(),
    arguments_json: mt.type.string().nullable(), success: mt.type.integer().nullable(),
    outcome_status: mt.type.string().nullable(), output_status: mt.type.string().nullable(),
    result: mt.type.string().nullable(), error: mt.type.string().nullable(),
    exit_code: mt.type.integer().nullable(), duration_ms: mt.type.number().nullable(),
    truncated: mt.type.integer().nullable(), full_bytes: mt.type.integer().nullable(),
    full_hash: mt.type.string().nullable(), full_reference: mt.type.string().nullable(),
    framework_metadata_json: mt.type.string().nullable(), raw_json: mt.type.string().nullable(),
  });
};

exports.fileRows = function(sessionId = null) {
  return mt.sql`
    SELECT
      session_id,
      tool_call_id,
      target_ordinal,
      path,
      native_path,
      operation_type,
      tool_name,
      evidence_kind,
      evidence_status,
      success,
      cwd,
      resolved,
      source_reference,
      turn_index
    FROM files
    WHERE (${sessionId} IS NULL OR session_id = ${sessionId})
    ORDER BY session_id, tool_call_id, target_ordinal
  `;
};

exports.eventRows = function(sessionId = null) {
  return mt.sql`
    SELECT
      session_id,
      event_id,
      turn_index,
      ordinal,
      kind,
      role,
      tool_call_id,
      annotation_id,
      title,
      summary,
      text,
      severity,
      collapsed_by_default,
      raw_json
    FROM events
    WHERE (${sessionId} IS NULL OR session_id = ${sessionId})
    ORDER BY COALESCE(turn_index, 999999), COALESCE(ordinal, 999999), event_id
  `;
};

exports.transcriptRows = function({sessionId = null, includeTools = false} = {}) {
  return mt.sql`
    SELECT * FROM (
      SELECT 'turn-' || turn_index AS id, session_id, turn_index, 0 AS ordinal, role, 'message' AS kind, model AS name, role || ' message' AS title, content AS text, timestamp,
          COALESCE(input_tokens, 0) + COALESCE(output_tokens, 0) + COALESCE(reasoning_tokens, 0) AS tokens,
          NULL AS tool_call_id, 'info' AS severity, 0 AS collapsed_by_default, raw_json AS metadata, CASE WHEN input_tokens IS NOT NULL OR output_tokens IS NOT NULL OR reasoning_tokens IS NOT NULL THEN 'recorded_usage' ELSE 'unknown' END AS token_origin, 'tokens' AS token_unit, 'recorded_fields_zero_fill' AS token_method
        FROM turns WHERE (${sessionId} IS NULL OR session_id = ${sessionId}) AND COALESCE(content, '') != ''
      UNION ALL
      SELECT 'turn-' || turn_index || '-thinking' AS id, session_id, turn_index, 1 AS ordinal, role, 'thinking' AS kind, model AS name, 'Thinking' AS title, thinking AS text, timestamp,
          COALESCE(reasoning_tokens, 0) AS tokens, NULL AS tool_call_id, 'info' AS severity, 1 AS collapsed_by_default, raw_json AS metadata, CASE WHEN reasoning_tokens IS NOT NULL THEN 'recorded_usage' ELSE 'unknown' END AS token_origin, 'tokens' AS token_unit, 'recorded_reasoning_zero_fill' AS token_method
        FROM turns WHERE (${sessionId} IS NULL OR session_id = ${sessionId}) AND COALESCE(thinking, '') != ''
      UNION ALL
      SELECT 'tool-' || tc.tool_call_id AS id, tc.session_id, COALESCE(ttc.turn_index, tc.emitting_turn_index) AS turn_index, 10 + COALESCE(ttc.ordinal, 0) AS ordinal, 'tool' AS role,
          CASE WHEN COALESCE(tc.success, 1) = 0 THEN 'tool_error' ELSE 'tool_result' END AS kind,
          tc.tool_name AS name, tc.tool_name || ' / ' || COALESCE(tc.operation_type, 'operation') AS title,
          COALESCE(tc.error, tc.result, tc.command, tc.file_path, '') AS text, tc.timestamp,
          CAST(ROUND(COALESCE(tc.full_bytes, LENGTH(CAST(COALESCE(tc.result, tc.error, '') AS BLOB))) / 4.0) AS INTEGER) AS tokens,
          tc.tool_call_id, CASE WHEN COALESCE(tc.success, 1) = 0 THEN 'error' ELSE 'info' END AS severity, 1 AS collapsed_by_default, tc.raw_json AS metadata, CASE WHEN tc.full_bytes IS NOT NULL OR tc.result IS NOT NULL OR tc.error IS NOT NULL THEN 'estimated' ELSE 'unknown' END AS token_origin, 'estimated_tokens' AS token_unit, 'round_utf8_bytes_div_4' AS token_method
        FROM tool_calls tc LEFT JOIN (
          SELECT session_id, tool_call_id, CASE WHEN COUNT(DISTINCT turn_index)=1 THEN MIN(turn_index) END AS turn_index, MIN(ordinal) AS ordinal
          FROM turn_tool_calls GROUP BY session_id, tool_call_id
        ) ttc ON ttc.session_id = tc.session_id AND ttc.tool_call_id = tc.tool_call_id
        WHERE ${includeTools} AND (${sessionId} IS NULL OR tc.session_id = ${sessionId})
    )
    ORDER BY session_id, turn_index IS NULL, turn_index, ordinal, id
  `;
};

exports.timelineRows = function(sessionId = null) {
  return mt.sql`
    WITH membership AS (
     SELECT DISTINCT session_id, turn_index, tool_call_id FROM turn_tool_calls
    ), tool_stats AS (
     SELECT m.session_id, m.turn_index, COUNT(*) AS tool_call_count,
     SUM(CASE WHEN tc.success=0 THEN 1 ELSE 0 END) AS failed_tool_count,
     SUM(CASE WHEN tc.success IS NULL THEN 1 ELSE 0 END) AS unknown_tool_count
     FROM membership m JOIN tool_calls tc ON tc.session_id=m.session_id AND tc.tool_call_id=m.tool_call_id
     GROUP BY m.session_id, m.turn_index
    ), file_stats AS (
     SELECT m.session_id, m.turn_index, COUNT(DISTINCT f.path) AS file_count,
     COUNT(*) AS file_touch_count,
     SUM(CASE WHEN f.evidence_status='confirmed' AND f.success=1 THEN 1 ELSE 0 END) AS confirmed_file_target_count
     FROM membership m JOIN files f ON f.session_id=m.session_id AND f.tool_call_id=m.tool_call_id
     GROUP BY m.session_id, m.turn_index
    )
    SELECT t.session_id, t.turn_index, t.role, t.timestamp, SUBSTR(COALESCE(t.content, t.thinking, ''), 1, 600) AS preview,
      COALESCE(t.input_tokens, 0) + COALESCE(t.output_tokens, 0) + COALESCE(t.cache_read_tokens, 0) + COALESCE(t.cache_creation_tokens, 0) + COALESCE(t.reasoning_tokens, 0) + COALESCE(t.tool_tokens, 0) AS total_tokens,
      CASE WHEN t.input_tokens IS NOT NULL OR t.output_tokens IS NOT NULL OR t.cache_read_tokens IS NOT NULL OR t.cache_creation_tokens IS NOT NULL OR t.reasoning_tokens IS NOT NULL OR t.tool_tokens IS NOT NULL THEN 'recorded_usage' ELSE 'unknown' END AS token_origin, 'tokens' AS token_unit, 'recorded_fields_zero_fill' AS token_method,
      COALESCE(ts.tool_call_count,0) AS tool_call_count, COALESCE(ts.failed_tool_count,0) AS failed_tool_count,
      COALESCE(ts.unknown_tool_count,0) AS unknown_tool_count, COALESCE(fs.file_count,0) AS file_count,
      COALESCE(fs.file_touch_count,0) AS file_touch_count, COALESCE(fs.confirmed_file_target_count,0) AS confirmed_file_target_count,
      CASE WHEN COALESCE(t.thinking, '') != '' THEN 1 ELSE 0 END AS has_thinking,
      EXISTS(SELECT 1 FROM annotations a WHERE a.session_id=t.session_id AND a.scope_type='turn' AND a.target_id=CAST(t.turn_index AS TEXT)) AS has_annotations
    FROM turns t
    LEFT JOIN tool_stats ts ON ts.session_id=t.session_id AND ts.turn_index=t.turn_index
    LEFT JOIN file_stats fs ON fs.session_id=t.session_id AND fs.turn_index=t.turn_index
    WHERE (${sessionId} IS NULL OR t.session_id = ${sessionId})
    ORDER BY t.turn_index
  `;
};

const tokenColumns = function() {
  const {string, integer} = mt.type;
  return {
    scope: string(), session_id: string(), turn_index: integer().nullable(),
    role: string().nullable(), tool_call_id: string().nullable(), input_tokens: integer(),
    output_tokens: integer(), cache_read_tokens: integer(), cache_creation_tokens: integer(),
    reasoning_tokens: integer(), tool_tokens: integer(), total_tokens: integer(),
    reported_usage_fields: integer(), token_origin: string(), token_unit: string(),
    token_method: string(),
  };
};

const groupedTokenUsage = function(sessionId, byRole) {
  return mt.sql`
    WITH usage AS (
      SELECT session_id, CASE WHEN ${byRole} THEN role END AS group_role,
        COALESCE(input_tokens, 0) AS input_tokens,
        COALESCE(output_tokens, 0) AS output_tokens,
        COALESCE(cache_read_tokens, 0) AS cache_read_tokens,
        COALESCE(cache_creation_tokens, 0) AS cache_creation_tokens,
        COALESCE(reasoning_tokens, 0) AS reasoning_tokens,
        COALESCE(tool_tokens, 0) AS tool_tokens,
        (input_tokens IS NOT NULL) + (output_tokens IS NOT NULL) +
          (cache_read_tokens IS NOT NULL) + (cache_creation_tokens IS NOT NULL) +
          (reasoning_tokens IS NOT NULL) + (tool_tokens IS NOT NULL) AS reported_usage_fields
      FROM turns WHERE (${sessionId} IS NULL OR session_id = ${sessionId})
    )
    SELECT ${byRole ? "role" : "session"} AS scope, session_id, NULL AS turn_index,
      group_role AS role, NULL AS tool_call_id, SUM(input_tokens) AS input_tokens,
      SUM(output_tokens) AS output_tokens, SUM(cache_read_tokens) AS cache_read_tokens,
      SUM(cache_creation_tokens) AS cache_creation_tokens, SUM(reasoning_tokens) AS reasoning_tokens,
      SUM(tool_tokens) AS tool_tokens,
      SUM(input_tokens + output_tokens + cache_read_tokens + cache_creation_tokens +
        reasoning_tokens + tool_tokens) AS total_tokens,
      SUM(reported_usage_fields) AS reported_usage_fields,
      CASE WHEN SUM(reported_usage_fields) > 0 THEN 'recorded_usage' ELSE 'unknown' END AS token_origin,
      'tokens' AS token_unit, 'sum_recorded_fields_zero_fill' AS token_method
    FROM usage
    GROUP BY session_id, group_role
    ORDER BY session_id, total_tokens DESC
  `.columns(tokenColumns());
};

exports.sessionTokenUsage = (sessionId = null) => groupedTokenUsage(sessionId, false);
exports.roleTokenUsage = (sessionId = null) => groupedTokenUsage(sessionId, true);

exports.toolTokenUsage = function(sessionId = null) {
  return mt.sql`
    SELECT 'tool' AS scope, session_id, emitting_turn_index AS turn_index, 'tool' AS role, tool_call_id, 0 AS input_tokens, 0 AS output_tokens, 0 AS cache_read_tokens, 0 AS cache_creation_tokens, 0 AS reasoning_tokens, 0 AS tool_tokens, CAST(ROUND(COALESCE(full_bytes, LENGTH(CAST(COALESCE(result, error, '') AS BLOB))) / 4.0) AS INTEGER) AS total_tokens, 0 AS reported_usage_fields, CASE WHEN full_bytes IS NOT NULL OR result IS NOT NULL OR error IS NOT NULL THEN 'estimated' ELSE 'unknown' END AS token_origin, 'estimated_tokens' AS token_unit, 'round_utf8_bytes_div_4' AS token_method
    FROM tool_calls
    WHERE (${sessionId} IS NULL OR session_id = ${sessionId})
    ORDER BY session_id, turn_index, tool_call_id
  `.columns(tokenColumns());
};

exports.turnTokenUsage = function(sessionId = null) {
  return mt.sql`
    SELECT 'turn' AS scope, session_id, turn_index, role, NULL AS tool_call_id, COALESCE(input_tokens, 0) AS input_tokens, COALESCE(output_tokens, 0) AS output_tokens, COALESCE(cache_read_tokens, 0) AS cache_read_tokens, COALESCE(cache_creation_tokens, 0) AS cache_creation_tokens, COALESCE(reasoning_tokens, 0) AS reasoning_tokens, COALESCE(tool_tokens, 0) AS tool_tokens,
      COALESCE(input_tokens, 0) + COALESCE(output_tokens, 0) + COALESCE(cache_read_tokens, 0) + COALESCE(cache_creation_tokens, 0) + COALESCE(reasoning_tokens, 0) + COALESCE(tool_tokens, 0) AS total_tokens,
      (input_tokens IS NOT NULL) + (output_tokens IS NOT NULL) + (cache_read_tokens IS NOT NULL) + (cache_creation_tokens IS NOT NULL) + (reasoning_tokens IS NOT NULL) + (tool_tokens IS NOT NULL) AS reported_usage_fields, CASE WHEN ((input_tokens IS NOT NULL) + (output_tokens IS NOT NULL) + (cache_read_tokens IS NOT NULL) + (cache_creation_tokens IS NOT NULL) + (reasoning_tokens IS NOT NULL) + (tool_tokens IS NOT NULL))>0 THEN 'recorded_usage' ELSE 'unknown' END AS token_origin, 'tokens' AS token_unit, 'recorded_fields_zero_fill' AS token_method
    FROM turns
    WHERE (${sessionId} IS NULL OR session_id = ${sessionId})
    ORDER BY session_id, turn_index
  `.columns(tokenColumns());
};

// A table-rooted count keeps sessions that have no tool rows.
exports.sessionToolCounts = function(db, sessionId = null) {
  const sessions = mt.from(db.table("sessions"));
  const tools = mt.from(db.table("tool_calls"));
  const selected = sessionId === null ? sessions : sessions.where(s => s.session_id.eq(sessionId));
  return selected.select(s => ({
    session_id: s.session_id,
    tool_count: tools.where(t => t.session_id.eq(s.session_id)).count(),
  })).orderBy(s => s.session_id.asc());
};

const completeRows = function(result) {
  if (result.error || !result.complete) {
    throw Error(result.error || "Incomplete query result");
  }
  return result.rows;
};

// Frames group two complete result sets by session and associated turn.
exports.turnFrames = function(db, sessionId = null) {
  const blocks = completeRows(db.execute(exports.eventRows(sessionId).build()));
  const tools = completeRows(db.execute(exports.toolRows(sessionId).build()));
  const frames = new Map();
  const frameFor = (row) => {
    const key = JSON.stringify([row.session_id, row.turn_index]);
    if (!frames.has(key)) {
      frames.set(key, {
        sessionId: row.session_id, turnIndex: row.turn_index,
        unassociated: row.turn_index === null, blocks: [], toolCalls: [],
        stats: {chars: 0, toolCalls: 0, failedToolCalls: 0, unknownToolCalls: 0, estimatedTokens: 0},
      });
    }
    return frames.get(key);
  };
  for (const block of blocks) frameFor(block).blocks.push(block);
  for (const tool of tools) {
    const frame = frameFor(tool);
    frame.toolCalls.push(tool);
    frame.stats.toolCalls++;
    if (tool.success === false || tool.success === 0) {
      frame.stats.failedToolCalls++;
    } else if (tool.success !== true && tool.success !== 1) {
      frame.stats.unknownToolCalls++;
    }
  }
  return [...frames.values()].sort((left, right) => {
    if (left.sessionId !== right.sessionId) return left.sessionId < right.sessionId ? -1 : 1;
    if (left.unassociated !== right.unassociated) return left.unassociated ? 1 : -1;
    return left.turnIndex - right.turnIndex;
  });
};
