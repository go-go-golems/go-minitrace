function _db(mt, opts) {
  let b = mt.db().RuntimeArchives().SQLiteMemory();
  if (opts && opts.cellBytes) b = b.MaxCellBytes(opts.cellBytes);
  return b.Build();
}

function builderRecipe(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  try {
    const result = db.execute(mt.query().ToolRows().Build());
    return [{
      envelope_keys: Object.keys(result),
      complete: result.complete,
      count: result.count,
      error: result.error || null,
      rows: (result.rows || []).map(r => ({
        tool_call_id: r.tool_call_id,
        source_ordinal: r.source_ordinal,
        arguments_json: r.arguments_json,
        output_status: r.output_status,
        outcome_status: r.outcome_status
      }))
    }];
  } finally {
    db.close();
  }
}

function boundArgs(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  try {
    const result = db.queryResult(
      "SELECT tool_call_id, tool_name, source_ordinal FROM tool_calls WHERE tool_name = ? ORDER BY source_ordinal",
      "synthetic_read");
    return [{ rows: result.rows, count: result.count, complete: result.complete }];
  } finally {
    db.close();
  }
}

function exceptions(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  const out = [];
  function attempt(label, fn) {
    try {
      const v = fn();
      out.push({ label, threw: false, value: v });
    } catch (e) {
      out.push({ label, threw: true, error: String(e) });
    }
  }
  try {
    attempt("query_invalid_sql", () => db.query("SELEC nope FROM"));
    attempt("queryOne_invalid_sql", () => db.queryOne("SELEC nope FROM"));
    attempt("query_disallowed_table", () => db.query("SELECT * FROM sqlite_master"));
    attempt("queryResult_invalid_sql", () => {
      const r = db.queryResult("SELEC nope FROM");
      return { error: r.error, count: r.count, complete: r.complete, rows: r.rows };
    });
    attempt("execute_invalid_sql", () => {
      const r = db.execute({ sql: "SELEC nope FROM" });
      return { error: r.error, count: r.count, complete: r.complete, rows: r.rows };
    });
    attempt("execute_bound_args", () => {
      const r = db.execute({ sql: "SELECT ? AS v", args: ["abé"] });
      return { rows: r.rows, complete: r.complete, error: r.error || null };
    });
    attempt("queryOk", () => db.query("SELECT COUNT(*) AS n FROM tool_calls"));
    return out;
  } finally {
    db.close();
  }
}

function toolFidelity(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  try {
    const cols = `tool_call_id, source_ordinal, arguments_json, output_status, outcome_status, success, timestamp, emitting_turn_index`;
    const archiveOrder = db.query(
      "SELECT " + cols + " FROM tool_calls ORDER BY source_ordinal");
    const chronoOrder = db.query(
      "SELECT " + cols + " FROM tool_calls ORDER BY timestamp, tool_call_id");
    const recipeDefault = db.query(mt.query().ToolRows().Build().sql(),
      ...mt.query().ToolRows().Build().args());
    return [{
      archive_order: archiveOrder,
      chronological_order: chronoOrder,
      recipe_default_order: recipeDefault.map(r => r.tool_call_id),
      recipe_is_archive_order: JSON.stringify(recipeDefault.map(r => r.tool_call_id)) ===
        JSON.stringify(archiveOrder.map(r => r.tool_call_id))
    }];
  } finally {
    db.close();
  }
}

function tokenProvenance(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  try {
    const turnRows = db.execute(mt.query().TokenUsageRows().ByTurn().Build());
    const toolRows = db.execute(mt.query().TokenUsageRows().ByTool().Build());
    const projected = (r) => ({
      scope: r.scope, turn_index: r.turn_index, role: r.role,
      tool_call_id: r.tool_call_id,
      input_tokens: r.input_tokens, output_tokens: r.output_tokens,
      total_tokens: r.total_tokens, reported_usage_fields: r.reported_usage_fields,
      token_origin: r.token_origin, token_unit: r.token_unit, token_method: r.token_method
    });
    return [{
      turn_rows: turnRows.rows.map(projected),
      tool_rows: toolRows.rows.map(projected),
      turn_complete: turnRows.complete,
      tool_complete: toolRows.complete
    }];
  } finally {
    db.close();
  }
}

function corpusManifestFixtures(filters) {
  const mt = require("minitrace");
  const paths = [
    "/private/tmp/gmt015-persona/inputs/archive.json",
    "/private/tmp/gmt015-persona/inputs/archive-copy.json",
    "/private/tmp/gmt015-persona/inputs/archive-conflict.json"
  ];
  const m = mt.corpusManifest(paths);
  return [{
    duplicateCount: m.duplicateCount,
    conflictCount: m.conflictCount,
    entries: m.entries.map((e, i) => ({
      index: i, path: e.path, sessionId: e.sessionId,
      duplicateOf: e.duplicateOf, conflict: e.conflict,
      entry_keys: Object.keys(e), sha256: e.sha256 || null,
      sourceFingerprint: e.sourceFingerprint || null,
      sizeBytes: e.sizeBytes
    })),
    manifest_keys: Object.keys(m),
    groups: (m.groups || []).map(g => ({
      sessionId: g.sessionId, entryIndexes: g.entryIndexes,
      contentVariants: g.contentVariants, conflict: g.conflict
    })),
    sessions: m.sessions
  }];
}

function realSummary(filters) {
  const mt = require("minitrace");
  const db = _db(mt);
  try {
    const frameworks = db.query(
      "SELECT agent_framework, COUNT(*) AS sessions, SUM(turn_count) AS turns, SUM(tool_call_count) AS tool_calls FROM sessions GROUP BY agent_framework");
    const tools = db.query(
      "SELECT s.agent_framework, tc.tool_name, COUNT(*) AS n, SUM(CASE WHEN tc.outcome_status='failed' THEN 1 ELSE 0 END) AS failed FROM tool_calls tc JOIN sessions s ON s.session_id = tc.session_id GROUP BY s.agent_framework, tc.tool_name ORDER BY n DESC LIMIT 8");
    const tokens = db.query(
      "SELECT s.agent_framework, SUM(m.total_input_tokens) AS recorded_input, SUM(m.total_output_tokens) AS recorded_output, SUM(m.total_cache_read_tokens) AS recorded_cache_read, SUM(m.total_reasoning_tokens) AS recorded_reasoning, SUM(m.total_tool_tokens) AS recorded_tool FROM metrics m JOIN sessions s ON s.session_id = m.session_id GROUP BY s.agent_framework");
    const turnProv = db.query(
      "SELECT s.agent_framework, CASE WHEN t.input_tokens IS NULL AND t.output_tokens IS NULL AND t.cache_read_tokens IS NULL AND t.cache_creation_tokens IS NULL AND t.reasoning_tokens IS NULL AND t.tool_tokens IS NULL THEN 'unknown' ELSE 'recorded_usage' END AS token_origin, COUNT(*) AS n FROM turns t JOIN sessions s ON s.session_id = t.session_id GROUP BY 1, 2 ORDER BY 1, 2");
    const toolEstimate = db.query(
      "SELECT s.agent_framework, COUNT(*) AS tool_rows, SUM(CAST(ROUND(COALESCE(tc.full_bytes, LENGTH(CAST(COALESCE(tc.result, tc.error, '') AS BLOB))) / 4.0) AS INTEGER)) AS estimated_tokens_sum FROM tool_calls tc JOIN sessions s ON s.session_id = tc.session_id GROUP BY s.agent_framework");
    return [{ frameworks, tools, recorded_token_totals: tokens, turn_token_origin: turnProv, tool_estimate_units: toolEstimate }];
  } finally {
    db.close();
  }
}

__verb__("builderRecipe", { name: "builder-recipe", short: "Builder recipe envelope", fields: {} });
__verb__("boundArgs", { name: "bound-args", short: "Bound-arg SQL recipe", fields: {} });
__verb__("exceptions", { name: "exceptions", short: "Query/queryOne vs envelopes", fields: {} });
__verb__("toolFidelity", { name: "tool-fidelity", short: "Tool provenance orderings", fields: {} });
__verb__("tokenProvenance", { name: "token-provenance", short: "Token provenance", fields: {} });
__verb__("corpusManifestFixtures", { name: "manifest-fixtures", short: "Corpus manifest over fixture archives", fields: {} });
__verb__("realSummary", { name: "real-summary", short: "Real archive summary", fields: {} });
