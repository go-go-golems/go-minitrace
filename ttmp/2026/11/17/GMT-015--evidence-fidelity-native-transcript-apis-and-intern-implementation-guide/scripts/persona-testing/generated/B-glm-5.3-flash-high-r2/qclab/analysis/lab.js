// B-glm-5.3-flash-high-r2 evidence lab: unicode bytes, clipping, strict mode,
// row overflow, builder recipes vs bound SQL, envelope contrast, corpusManifest,
// token provenance, per-framework summaries.

function _src(paths) {
  const mt = require("minitrace");
  const s = mt.sources();
  for (const p of paths) s.File(p);
  return s.Build();
}

function _dbFrom(paths, extra) {
  const mt = require("minitrace");
  let db = mt.db().Sources(_src(paths)).SQLiteMemory();
  if (extra) db = extra(db);
  return db.Build();
}

// ---- verb 1: unicode byte size + permissive clipping at 3 bytes ------------
function unicodeChecks() {
  const mt = require("minitrace");
  const out = { utf8ByteLength_abe: mt.utf8ByteLength("abé") };
  const db = _dbFrom(["/private/tmp/gmt015-persona/inputs/archive.json"], (d) => d.MaxCellBytes(3));
  try {
    const permissive = db.queryResult("SELECT 'abé' AS text");
    out.permissive = {
      columns: permissive.columns,
      rows: permissive.rows,
      count: permissive.count,
      complete: permissive.complete,
      truncated: permissive.truncated,
      error: permissive.error || null,
      cellDiagnostics: permissive.cellDiagnostics || null,
    };
  } finally { db.close(); }
  return out;
}

// ---- verb 2: strict rejection, no partial rows -----------------------------
function strictChecks() {
  const db = _dbFrom(["/private/tmp/gmt015-persona/inputs/archive.json"], (d) => d.MaxCellBytes(3).StrictEvidence(true));
  try {
    const strict = db.queryResult("SELECT 'abé' AS text");
    return {
      complete: strict.complete,
      error: strict.error || null,
      rows: strict.rows,
      count: strict.count,
      truncated: strict.truncated || null,
      cellDiagnostics: strict.cellDiagnostics || null,
    };
  } finally { db.close(); }
}

// ---- verb 3: row overflow separately ---------------------------------------
function rowOverflow() {
  const db = _dbFrom(["/private/tmp/gmt015-persona/inputs/archive.json"], (d) => d.MaxRows(1));
  try {
    const r = db.queryResult("SELECT 1 AS n UNION ALL SELECT 2 ORDER BY n");
    return {
      columns: r.columns, rows: r.rows, count: r.count,
      complete: r.complete, truncated: r.truncated || null,
      rowsTruncated: r.rowsTruncated === undefined ? null : r.rowsTruncated,
      error: r.error || null, cellDiagnostics: r.cellDiagnostics || null,
    };
  } finally { db.close(); }
}

// ---- verb 4: envelope contrast ---------------------------------------------
// builder recipe + bound-argument SQL + invalid SQL through query/queryOne
// (throwing) vs queryResult/execute (envelope).
function envelopeContrast(filters) {
  const mt = require("minitrace");
  const out = {};
  const db = _dbFrom(["/private/tmp/gmt015-persona/inputs/archive.json"]);
  try {
    const first = db.queryOne("SELECT session_id FROM sessions LIMIT 1");
    const sid = (filters && filters.session_id) || (first ? first.session_id : null);

    // builder recipe
    const recipe = mt.query().ToolRows().Build();
    try {
      const exec = db.execute(recipe);
      out.builder_recipe = {
        sql_head: recipe.sql().slice(0, 160),
        args: recipe.args(),
        count: exec.count,
        complete: exec.complete,
        error: exec.error || null,
      };
    } catch (e) {
      out.builder_recipe = { threw: String(e).slice(0, 200) };
    }

    // bound-argument SQL
    const bound = db.queryResult("SELECT session_id, agent_framework FROM sessions WHERE session_id = ?", sid);
    out.bound_sql = { sql: "SELECT session_id, agent_framework FROM sessions WHERE session_id = ?", arg: sid, rows: bound.rows, count: bound.count, complete: bound.complete, error: bound.error || null };

    // invalid SQL: envelope path
    const bad = db.queryResult("SELECT no_such_col FROM no_such_table");
    out.invalid_queryResult = { complete: bad.complete, error: bad.error || null, rows: bad.rows, count: bad.count };

    // invalid SQL: throwing convenience paths
    const threw = {};
    try { db.query("SELECT no_such_col FROM no_such_table"); threw.query = "no-throw"; }
    catch (e) { threw.query = String(e).slice(0, 200); }
    try { db.queryOne("SELECT no_such_col FROM no_such_table"); threw.queryOne = "no-throw"; }
    catch (e) { threw.queryOne = String(e).slice(0, 200); }
    out.throwing_paths = threw;
  } finally { db.close(); }
  return out;
}

// ---- verb 5: tool ordinal / arguments / status fidelity --------------------
function toolFidelity(filters) {
  const mt = require("minitrace");
  const db = _dbFrom([filters.path]);
  try {
    const recipe = mt.query().ToolRows().Build();
    const base = db.execute(recipe);
    if (base.error) return { error: base.error };
    const cols = base.columns;
    const rows = base.rows.map((row) => {
      const o = {};
      cols.forEach((c, i) => { o[c] = row[i]; });
      return o;
    });
    const fields = ["tool_call_id", "source_ordinal", "timestamp", "arguments_json", "output_status", "outcome_status", "success", "turn_index"];
    const slim = rows.map((r) => {
      const o = {};
      for (const f of fields) o[f] = r[f] === undefined ? null : r[f];
      return o;
    });
    const fieldIdx = {};
    for (const f of fields) fieldIdx[f] = cols.indexOf(f);
    // archive order vs chronological order
    const byArchive = rows.slice().sort((a, b) => a.source_ordinal - b.source_ordinal).map((r) => r.tool_call_id);
    const byChrono = rows.slice().sort((a, b) => String(a.timestamp).localeCompare(String(b.timestamp))).map((r) => r.tool_call_id);
    const statusPairs = {};
    for (const r of rows) {
      const k = String(r.output_status) + "|" + String(r.outcome_status);
      statusPairs[k] = (statusPairs[k] || 0) + 1;
    }
    return {
      columns: cols, field_column_index: fieldIdx,
      rows: slim.slice(0, 12), row_count: rows.length,
      archive_order_source_ordinal: byArchive,
      chronological_order_timestamp: byChrono,
      order_differs: JSON.stringify(byArchive) !== JSON.stringify(byChrono),
      output_status_vs_outcome_status_pairs: statusPairs,
    };
  } finally { db.close(); }
}

// ---- verb 6: token provenance ----------------------------------------------
function tokenProvenance(filters) {
  const db = _dbFrom([filters.path]);
  try {
    const recipe = mt.query().TokenUsageRows().Build();
    const r = db.execute(recipe);
    if (r.error) return { error: r.error };
    const cols = r.columns;
    const rows = r.rows.map((row) => {
      const o = {};
      cols.forEach((c, i) => { o[c] = row[i]; });
      return o;
    });
    const wanted = ["token_origin", "token_unit", "token_method", "estimated_tokens", "reported_usage_fields", "input_tokens", "output_tokens", "total_tokens", "tool_call_id", "turn_index"];
    const slim = rows.map((row) => {
      const o = {};
      for (const f of wanted) o[f] = row[f] === undefined ? null : row[f];
      return o;
    });
    // provenance breakdown without summing incompatible units
    const byOrigin = {};
    for (const row of slim) {
      const k = String(row.token_origin);
      byOrigin[k] = (byOrigin[k] || 0) + 1;
      if (!byOrigin[k + "_units"]) byOrigin[k + "_units"] = [];
      if (!byOrigin[k + "_units"].includes(row.token_unit)) byOrigin[k + "_units"].push(row.token_unit);
    }
const estRows = rows.filter((x) => x.estimated_tokens !== null && x.estimated_tokens !== undefined).slice(0, 5);
    const toolRows = rows.filter((x) => x.tool_call_id).slice(0, 5);
    return { columns: cols, rows: slim.slice(0, 20), row_count: rows.length, provenance_counts: byOrigin, estimate_rows: estRows, tool_level_rows: toolRows };
  } finally { db.close(); }
}

// ---- verb 7: corpus manifest on the three fixture archives -----------------
function corpusManifest() {
  const paths = [
    "/private/tmp/gmt015-persona/inputs/archive.json",
    "/private/tmp/gmt015-persona/inputs/archive-copy.json",
    "/private/tmp/gmt015-persona/inputs/archive-conflict.json",
  ];
  const m = require("minitrace").corpusManifest(paths);
  return {
    entries: m.entries,
    duplicateCount: m.duplicateCount,
    conflictCount: m.conflictCount,
    sessions: m.sessions,
  };
}

// ---- verb 8: per-framework token/tool summary on one real archive ----------
function frameworkSummary(filters) {
  const db = _dbFrom([filters.path]);
  try {
    const sess = db.queryOne("SELECT session_id, agent_framework, model, turn_count, tool_call_count FROM sessions LIMIT 1") || {};
    const tools = db.queryOne("SELECT COUNT(*) AS n FROM tool_calls") || {};
    const tokRecipe = mt.query().TokenUsageRows().Build();
    const tokRes = db.execute(tokRecipe);
    const byOrigin = {};
    if (!tokRes.error) {
      for (const r of tokRes.rows) {
        const k = r.token_origin + "/" + r.token_unit;
        byOrigin[k] = (byOrigin[k] || 0) + 1;
      }
    }
    const cache = db.cacheInfo ? db.cacheInfo() : null;
    return { session: sess, tool_rows: tools.n, token_breakdown_by_origin_unit: byOrigin, token_error: tokRes.error || null, cacheInfo: cache };
  } finally { db.close(); }
}


// ---- verb 9: tool-scope estimated token provenance -------------------------
function tokenProvenanceTools(filters) {
  const mt = require("minitrace");
  const db = _dbFrom([filters.path]);
  try {
    const recipe = mt.query().TokenUsageRows().ByTool().Build();
    const r = db.execute(recipe);
    if (r.error) return { error: r.error };
    const rows = r.rows;
    const est = rows.filter((x) => x.estimated_tokens !== null && x.estimated_tokens !== undefined);
    const byOrigin = {};
    for (const x of rows) {
      const k = String(x.token_origin);
      byOrigin[k] = (byOrigin[k] || 0) + 1;
      if (!byOrigin[k + "_units"]) byOrigin[k + "_units"] = [];
      if (!byOrigin[k + "_units"].includes(x.token_unit)) byOrigin[k + "_units"].push(x.token_unit);
    }
    return { row_count: rows.length, provenance_counts: byOrigin, estimate_samples: est.slice(0, 4), note: "estimated rows use token_unit=estimated_tokens; must not be summed with token_unit=tokens" };
  } finally { db.close(); }
}

// ---- verb 10: explicit zero vs missing usage on the fixture ----------------
function tokenZeroVsMissing() {
  const mt = require("minitrace");
  const db = _dbFrom(["/private/tmp/gmt015-persona/inputs/archive.json"]);
  try {
    const recipe = mt.query().TokenUsageRows().ByTurn().Build();
    const r = db.execute(recipe);
    if (r.error) return { error: r.error };
    return { rows: r.rows, note: "turn 0 has no usage fields (unknown, zero-filled); turn 1 records explicit zeros with reported_usage_fields=2 (recorded_usage)" };
  } finally { db.close(); }
}

__verb__("unicodeChecks", { name: "unicode-checks", short: "utf8ByteLength + permissive 3-byte clip" });
__verb__("strictChecks", { name: "strict-checks", short: "strict evidence rejection envelope" });
__verb__("rowOverflow", { name: "row-overflow", short: "row overflow with MaxRows(1)" });
__verb__("envelopeContrast", { name: "envelope-contrast", short: "recipe + bound SQL + invalid SQL envelopes vs throws" });
__verb__("toolFidelity", { name: "tool-fidelity", short: "source_ordinal / arguments_json / status fields" });
__verb__("tokenProvenance", { name: "token-provenance", short: "token origin/unit/method provenance" });
__verb__("corpusManifest", { name: "corpus-manifest", short: "corpusManifest on three fixture archives" });
__verb__("frameworkSummary", { name: "framework-summary", short: "token/tool summary for one archive" });

__verb__("tokenProvenanceTools", { name: "token-provenance-tools", short: "tool-scope estimated token provenance" });
__verb__("tokenZeroVsMissing", { name: "token-zero-vs-missing", short: "explicit zero vs missing usage provenance" });
