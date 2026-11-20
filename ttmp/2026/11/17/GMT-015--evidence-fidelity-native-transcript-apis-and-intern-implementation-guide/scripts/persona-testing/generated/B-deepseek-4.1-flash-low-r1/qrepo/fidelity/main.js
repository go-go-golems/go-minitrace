// Evidence-fidelity probes over synthetic fixtures + corpus archives.
var FIX = "/private/tmp/gmt015-persona/inputs/";
var ARCH = FIX + "archive.json";

function _db(mt, files, opt) {
  var b = mt.db().Files(files).SQLiteMemory();
  if (opt && opt.maxCellBytes) b = b.MaxCellBytes(opt.maxCellBytes);
  if (opt && opt.maxRows) b = b.MaxRows(opt.maxRows);
  if (opt && opt.strict) b = b.StrictEvidence(true);
  return b.Build();
}

function _try(fn) {
  try { return { ok: true, value: fn() }; }
  catch (e) { return { ok: false, error: String(e) }; }
}

// 1. Unicode bytes, permissive clipping, strict rejection.
function unicode(filters) {
  const mt = require("minitrace");
  const db = _db(mt, [ARCH], { maxCellBytes: 3 });
  const rows = [];
  try {
    rows.push({ probe: "utf8ByteLength_abe", bytes: mt.utf8ByteLength("abé") });
    rows.push({ probe: "utf8ByteLength_ascii", bytes: mt.utf8ByteLength("abc") });
    const permissive = db.queryResult("SELECT 'abé' AS text");
    rows.push({
      probe: "permissive_clip",
      text: permissive.rows[0].text,
      count: permissive.count,
      complete: permissive.complete,
      truncated: permissive.truncated,
      originalBytes: permissive.cellDiagnostics[0].originalBytes,
      emittedBytes: permissive.cellDiagnostics[0].emittedBytes,
      cellTruncated: permissive.cellDiagnostics[0].truncated,
      validUtf8Clip: permissive.rows[0].text === "ab"
    });
  } finally { db.close(); }
  const sdb = _db(mt, [ARCH], { maxCellBytes: 3, strict: true });
  try {
    const strict = sdb.queryResult("SELECT 'abé' AS text");
    rows.push({
      probe: "strict_reject",
      hasError: typeof strict.error === "string" && strict.error.length > 0,
      error: strict.error,
      count: strict.count,
      rowCount: strict.rows.length,
      complete: strict.complete,
      truncated: strict.truncated
    });
  } finally { sdb.close(); }
  return rows;
}

// 2. Row overflow, independent from cell clipping.
function overflow(filters) {
  const mt = require("minitrace");
  const sql = "SELECT tool_call_id, result FROM tool_calls ORDER BY source_ordinal";
  const out = [];
  const full = _db(mt, [ARCH], {});
  try {
    const r = full.queryResult(sql);
    out.push({ probe: "no_row_limit", count: r.count, truncated: r.truncated, complete: r.complete, rowsTruncated: r.rowsTruncated === undefined ? null : r.rowsTruncated, error: r.error || null });
  } finally { full.close(); }
  const lim = _db(mt, [ARCH], { maxRows: 1 });
  try {
    const r = lim.queryResult(sql);
    out.push({ probe: "max_rows_1", count: r.count, truncated: r.truncated, complete: r.complete, rowsTruncated: r.rowsTruncated === undefined ? null : r.rowsTruncated, cellDiagnostics: r.cellDiagnostics ? r.cellDiagnostics.length : null, error: r.error || null });
  } finally { lim.close(); }
  const slim = _db(mt, [ARCH], { maxRows: 1, strict: true });
  try {
    const r = slim.queryResult(sql);
    out.push({ probe: "max_rows_1_strict", count: r.count, complete: r.complete, error: r.error || null });
  } finally { slim.close(); }
  return out;
}

// 3. Tool source_ordinal vs default chronological order; builder vs bound SQL;
//    query/queryOne exceptions and queryResult/execute envelopes incl. invalid SQL.
function tools(filters) {
  const mt = require("minitrace");
  const out = [];
  const db = _db(mt, [ARCH], {});
  try {
    const archiveOrder = db.query("SELECT tool_call_id, source_ordinal, timestamp FROM tool_calls ORDER BY source_ordinal");
    out.push({ probe: "archive_order", order: archiveOrder.map(r => r.tool_call_id).join(","),
      ordinals: archiveOrder.map(r => r.source_ordinal).join(","),
      timestamps: archiveOrder.map(r => r.timestamp).join(",") });
    const chrono = db.query("SELECT tool_call_id, source_ordinal, timestamp FROM tool_calls ORDER BY timestamp, tool_call_id");
    out.push({ probe: "chronological_order", order: chrono.map(r => r.tool_call_id).join(","),
      ordinals: chrono.map(r => r.source_ordinal).join(",") });

    // Builder recipe
    const recipe = mt.query().ToolRows().Build();
    const builderRows = db.query(recipe.sql(), ...recipe.args());
    out.push({ probe: "builder_recipe",
      recipeName: recipe.name(),
      ids: builderRows.map(r => r.tool_call_id).join(","),
      recipeDefaultOrder: builderRows.map(r => r.tool_call_id).join(","),
      arguments_json: builderRows.map(r => r.arguments_json).join(";"),
      output_status: builderRows.map(r => r.output_status).join(","),
      outcome_status: builderRows.map(r => r.outcome_status).join(","),
      success: builderRows.map(r => String(r.success)).join(",") });

    // Bound-argument SQL recipe (explicit ? args instead of string interpolation)
    const boundSql = "SELECT tool_call_id, source_ordinal, tool_name, arguments_json, output_status, outcome_status, success FROM tool_calls WHERE (? = '' OR session_id = ?) AND tool_name = ? ORDER BY source_ordinal";
    const boundRows = db.query(boundSql, "", "", "synthetic_read");
    out.push({ probe: "bound_sql_recipe", ids: boundRows.map(r => r.tool_call_id).join(","),
      ordinals: boundRows.map(r => r.source_ordinal).join(",") });
    const boundNone = db.query(boundSql, "", "", "no-such-tool");
    out.push({ probe: "bound_sql_no_match", count: boundNone.length });

    // query() throws on invalid SQL / queryOne throws when empty
    const qInv = _try(() => db.query("SELECT * FROM does_not_exist"));
    out.push({ probe: "query_invalid_sql", threw: !qInv.ok, error: qInv.ok ? null : qInv.error });
    const qOne = _try(() => db.queryOne("SELECT tool_call_id FROM tool_calls WHERE 1=0"));
    out.push({ probe: "queryOne_no_rows", threw: !qOne.ok,
      error: qOne.ok ? null : qOne.error,
      returnedType: qOne.ok ? typeof qOne.value : null,
      returnedJSON: qOne.ok ? JSON.stringify(qOne.value) : null });
    const qOneMany = _try(() => db.queryOne("SELECT tool_call_id FROM tool_calls ORDER BY source_ordinal"));
    out.push({ probe: "queryOne_many_rows", returned: qOneMany.ok ? qOneMany.value.tool_call_id : null });

    // queryResult / execute envelopes preserve errors instead of throwing
    const env1 = _try(() => db.queryResult("SELECT * FROM does_not_exist"));
    out.push({ probe: "queryResult_invalid_sql", threw: !env1.ok,
      envelopeError: env1.ok ? env1.value.error : null,
      count: env1.ok ? env1.value.count : null,
      complete: env1.ok ? env1.value.complete : null });
    const env2 = _try(() => db.execute({ sql: "SELECT * FROM does_not_exist", args: [] }));
    out.push({ probe: "execute_invalid_sql", threw: !env2.ok,
      envelopeError: env2.ok ? env2.value.error : null,
      count: env2.ok ? env2.value.count : null });
    const env3 = _try(() => db.execute(recipe));
    out.push({ probe: "execute_recipe", threw: !env3.ok,
      count: env3.ok ? env3.value.count : null,
      complete: env3.ok ? env3.value.complete : null,
      ids: env3.ok ? env3.value.rows.map(r => r.tool_call_id).join(",") : null });
    // builder/argument failure surfaces as thrown error, not envelope
    const env4 = _try(() => db.execute({ sql: "SELECT ? AS x", args: [] }));
    out.push({ probe: "execute_arg_mismatch", threw: !env4.ok,
      envelopeError: env4.ok ? (env4.value.error || null) : env4.error });
  } finally { db.close(); }
  return out;
}

// 4. Token provenance: missing vs explicit zero usage vs tool-output estimates.
function tokens(filters) {
  const mt = require("minitrace");
  const db = _db(mt, [ARCH], {});
  try {
    const recipe = mt.query().TokenUsageRows().ByTurn().Build();
    const rows = db.query(recipe.sql(), ...recipe.args());
    return rows.map(r => ({
      scope: r.scope, session_id: r.session_id, turn_index: r.turn_index, role: r.role,
      tool_call_id: r.tool_call_id, input_tokens: r.input_tokens, output_tokens: r.output_tokens,
      total_tokens: r.total_tokens, reported_usage_fields: r.reported_usage_fields,
      token_origin: r.token_origin, token_unit: r.token_unit, token_method: r.token_method
    }));
  } finally { db.close(); }
}

// 5. corpusManifest over the three corpus-identity fixture archives.
function manifest(filters) {
  const mt = require("minitrace");
  const m = mt.corpusManifest([FIX + "archive.json", FIX + "archive-copy.json", FIX + "archive-conflict.json"]);
  const entries = m.entries.map(e => ({
    file: e.path.split("/").pop(), sessionId: e.sessionId, sha256_12: String(e.sha256).slice(0, 12),
    sizeBytes: e.sizeBytes, duplicateOf: e.duplicateOf === undefined ? null : e.duplicateOf,
    conflict: e.conflict
  }));
  const groups = m.sessions.map(g => ({ sessionId: g.sessionId, entryIndexes: g.entryIndexes.join(","), contentVariants: g.contentVariants, conflict: g.conflict }));
  return [
    { probe: "manifest_totals", duplicateCount: m.duplicateCount, conflictCount: m.conflictCount, entryCount: m.entries.length, sessionGroupCount: m.sessions.length },
    ...entries.map(e => Object.assign({ probe: "entry" }, e)),
    ...groups.map(g => Object.assign({ probe: "session_group" }, g))
  ];
}

// 6. Token + tool summary for one real archive of a given framework.
function realSummary(filters) {
  const mt = require("minitrace");
  const path = filters.path;
  if (!path) throw new Error("--path is required");
  const db = _db(mt, [path], { maxRows: 100000 });
  const out = [];
  try {
    const s = db.queryOne("SELECT session_id, agent_framework, title, turn_count, tool_call_count FROM sessions");
    out.push({ probe: "session", session_id: s.session_id, framework: s.agent_framework, turns: s.turn_count, tools: s.tool_call_count, title: String(s.title).slice(0, 80) });
    const turns = db.query("SELECT COUNT(*) AS n, SUM(CASE WHEN token_origin='recorded_usage' THEN 1 ELSE 0 END) AS x FROM (SELECT CASE WHEN ((input_tokens IS NOT NULL)+(output_tokens IS NOT NULL)+(cache_read_tokens IS NOT NULL)+(cache_creation_tokens IS NOT NULL)+(reasoning_tokens IS NOT NULL)+(tool_tokens IS NOT NULL))>0 THEN 'recorded_usage' ELSE 'unknown' END AS token_origin FROM turns)");
    out.push({ probe: "turn_usage_origin", rows: turns[0].n, recorded_usage: turns[0].x, unknown: turns[0].n - turns[0].x });
    const sum = db.query("SELECT COALESCE(SUM(input_tokens),0) AS i, COALESCE(SUM(output_tokens),0) AS o, COALESCE(SUM(cache_read_tokens),0) AS cr, COALESCE(SUM(cache_creation_tokens),0) AS cc, COALESCE(SUM(reasoning_tokens),0) AS r FROM turns");
    out.push({ probe: "recorded_token_totals", input: sum[0].i, output: sum[0].o, cache_read: sum[0].cr, cache_creation: sum[0].cc, reasoning: sum[0].r, unit: "tokens", origin: "recorded_usage", note: "summed across turns only where fields non-null (COALESCE zero-fill hides missing)" });
    const est = db.query("SELECT COUNT(*) AS n, COALESCE(SUM(CAST(ROUND(COALESCE(full_bytes, LENGTH(CAST(COALESCE(result, error, '') AS BLOB))) / 4.0) AS INTEGER)),0) AS est FROM tool_calls");
    out.push({ probe: "tool_output_estimates", toolRows: est[0].n, estimated_tokens: est[0].est, unit: "estimated_tokens", method: "round_utf8_bytes_div_4", note: "incompatible with recorded token totals" });
    const tool = db.query("SELECT tool_name, COUNT(*) AS n, SUM(CASE WHEN success=0 THEN 1 ELSE 0 END) AS failed, SUM(CASE WHEN success IS NULL THEN 1 ELSE 0 END) AS unknown FROM tool_calls GROUP BY 1 ORDER BY n DESC LIMIT 10");
    tool.forEach(r => out.push({ probe: "tool", tool_name: r.tool_name, n: r.n, failed: r.failed, unknown: r.unknown }));
    const ord = db.query("SELECT COUNT(*) AS n FROM tool_calls WHERE source_ordinal IS NULL");
    out.push({ probe: "ordinal_null_count", n: ord[0].n });
  } finally { db.close(); }
  return out;
}

__section__("filters", {
  title: "Inputs",
  fields: { path: { type: "string", help: "Archive path for real-summary" } },
});

__verb__("unicode", { name: "unicode", short: "unicode/clip/strict probes", fields: {} });
__verb__("overflow", { name: "overflow", short: "row overflow probes", fields: {} });
__verb__("tools", { name: "tools", short: "ordinals + API envelopes", fields: {} });
__verb__("tokens", { name: "tokens", short: "token provenance rows", fields: {} });
__verb__("manifest", { name: "manifest", short: "corpusManifest identity", fields: {} });
__verb__("realSummary", { name: "real-summary", short: "real archive summary", fields: { filters: { bind: "filters" } } });
