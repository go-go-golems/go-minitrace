// Evidence-fidelity probes against the synthetic fixtures.
// Run with:
//   go-minitrace query commands --query-repository ./qrepo probe fidelity <verb> \
//     --archive-glob '/private/tmp/gmt015-persona/inputs/archive.json' --output json
const FIXTURES = "/private/tmp/gmt015-persona/inputs";

function _mt() {
  return require("minitrace");
}

function _file(name) {
  return FIXTURES + "/" + name;
}

function _try(fn) {
  try {
    return { threw: false, value: fn() };
  } catch (e) {
    return { threw: true, error: String(e && e.message ? e.message : e) };
  }
}

// --- exact byte size -------------------------------------------------------
function utf8() {
  const mt = _mt();
  return {
    text: "abé",
    utf8ByteLength: mt.utf8ByteLength("abé"),
    jsStringLength: "abé".length,
    codepoints: Array.from("abé").map((c) => c.codePointAt(0)),
  };
}

// --- permissive clipping at 3 bytes ---------------------------------------
function clip() {
  const mt = _mt();
  const db = mt.db().File(_file("archive.json")).SQLiteMemory()
    .MaxCellBytes(3).Build();
  try {
    const r = db.queryResult("SELECT 'abé' AS text");
    return {
      limit_bytes: 3,
      rows: r.rows,
      count: r.count,
      complete: r.complete,
      truncated: r.truncated,
      rowsTruncated: r.rowsTruncated,
      cellDiagnostics: r.cellDiagnostics,
      error: r.error === undefined ? null : r.error,
    };
  } finally {
    db.close();
  }
}

// --- strict rejection ------------------------------------------------------
function strict() {
  const mt = _mt();
  const db = mt.db().File(_file("archive.json")).SQLiteMemory()
    .MaxCellBytes(3).StrictEvidence(true).Build();
  try {
    const r = db.queryResult("SELECT 'abé' AS text");
    return {
      limit_bytes: 3,
      strict: true,
      rows: r.rows,
      rowCount: (r.rows || []).length,
      count: r.count,
      complete: r.complete,
      truncated: r.truncated,
      error: r.error === undefined ? null : r.error,
      cellDiagnostics: r.cellDiagnostics,
    };
  } finally {
    db.close();
  }
}

// --- row overflow (separate from cell clipping) ----------------------------
function overflow() {
  const mt = _mt();
  const db = mt.db().File(_file("archive.json")).SQLiteMemory()
    .MaxRows(2).MaxCellBytes(4000).Build();
  try {
    const r = db.queryResult(
      "SELECT tool_call_id FROM tool_calls ORDER BY tool_call_id");
    const lenient = {
      count: r.count,
      rows: (r.rows || []).length,
      rowIds: (r.rows || []).map((x) => x.tool_call_id),
      complete: r.complete,
      truncated: r.truncated,
      rowsTruncated: r.rowsTruncated,
      cellDiagnostics: r.cellDiagnostics,
      error: r.error === undefined ? null : r.error,
    };
    const strictDb = mt.db().File(_file("archive.json")).SQLiteMemory()
      .MaxRows(2).StrictEvidence(true).Build();
    let strictOverflow;
    try {
      const s = strictDb.queryResult(
        "SELECT tool_call_id FROM tool_calls ORDER BY tool_call_id");
      strictOverflow = {
        count: s.count,
        rows: (s.rows || []).length,
        complete: s.complete,
        truncated: s.truncated,
        error: s.error === undefined ? null : s.error,
      };
    } finally {
      strictDb.close();
    }
    return { limit_rows: 2, lenient, strict: strictOverflow };
  } finally {
    db.close();
  }
}

// --- tool row fidelity: source_ordinal / arguments_json / statuses --------
function toolrows() {
  const mt = _mt();
  const recipe = mt.query().ToolRows().Build();
  const db = mt.db().File(_file("archive.json")).SQLiteMemory()
    .MaxCellBytes(16000).Build();
  try {
    const archiveOrder = db.query(
      "SELECT source_ordinal, tool_call_id, timestamp, arguments_json, output_status, outcome_status, success, record_kind " +
      "FROM tool_calls ORDER BY source_ordinal");
    const chrono = db.query(
      "SELECT source_ordinal, tool_call_id, timestamp, output_status " +
      "FROM tool_calls ORDER BY timestamp, tool_call_id");
    const env = db.execute(recipe);
    return {
      recipe: { name: recipe.name(), sql: recipe.sql(), args: recipe.args() },
      recipeComplete: env.complete,
      recipeRows: (env.rows || []).map((r) => ({
        source_ordinal: r.source_ordinal,
        tool_call_id: r.tool_call_id,
        arguments_json: r.arguments_json,
        output_status: r.output_status,
        outcome_status: r.outcome_status,
      })),
      archiveOrder,
      archiveSourceOrdinals: archiveOrder.map((r) => r.source_ordinal),
      chronological: chrono,
      chronologicalSourceOrdinals: chrono.map((r) => r.source_ordinal),
    };
  } finally {
    db.close();
  }
}

// --- token provenance ------------------------------------------------------
function tokens() {
  const mt = _mt();
  const db = mt.db().File(_file("archive.json")).SQLiteMemory().Build();
  try {
    const turnRecipe = mt.query().TokenUsageRows().Build();
    const toolRecipe = mt.query().TokenUsageRows().ByTool().Build();
    return {
      turnRecipe: { name: turnRecipe.name(), sql: turnRecipe.sql(), args: turnRecipe.args() },
      turns: db.query(
        "SELECT turn_index, role, input_tokens, output_tokens, " +
        "(input_tokens IS NULL) AS input_null, (output_tokens IS NULL) AS output_null, " +
        "total_tokens, reported_usage_fields, token_origin, token_unit, token_method " +
        "FROM (" + turnRecipe.sql() + ") ORDER BY turn_index", ...turnRecipe.args()),
      toolRecipeSql: toolRecipe.sql(),
      toolEstimates: db.query(
        "SELECT tool_call_id, turn_index, total_tokens, token_origin, token_unit, token_method " +
        "FROM (" + toolRecipe.sql() + ") ORDER BY tool_call_id", ...toolRecipe.args()),
      rawTurnUsageCheck: db.query(
        "SELECT turn_index, json_extract(raw_json, '$.usage') AS usage_json FROM turns ORDER BY turn_index"),
    };
  } finally {
    db.close();
  }
}

// --- manifest identity preflight ------------------------------------------
function manifest() {
  const mt = _mt();
  const paths = [
    _file("archive.json"),
    _file("archive-copy.json"),
    _file("archive-conflict.json"),
  ];
  return mt.corpusManifest(paths);
}

// --- envelopes: query/queryOne throw, queryResult/execute return ----------
function envelopes() {
  const mt = _mt();
  const db = mt.db().File(_file("archive.json")).SQLiteMemory().Build();
  const out = {};
  try {
    out.query_invalid = _try(() => db.query("SELECT nope FROM missing_table"));
    out.queryOne_invalid = _try(() => db.queryOne("SELECT nope FROM missing_table"));
    out.queryResult_invalid = _try(() => db.queryResult("SELECT nope FROM missing_table"));
    out.execute_invalid = _try(() => db.execute({ sql: "SELECT nope FROM missing_table", args: [] }));
    out.queryResult_write = _try(() => db.queryResult("DELETE FROM tool_calls"));
    // bound-argument SQL recipe
    out.bound = db.execute({
      sql: "SELECT tool_call_id, source_ordinal FROM tool_calls WHERE output_status = ? ORDER BY source_ordinal",
      args: ["failed"],
    });
    // builder recipe vs raw SQL for the same shape
    out.builder = db.execute(mt.query().ToolRows().Build());
    out.recipe = {
      name: mt.query().ToolRows().Build().name(),
      sql: mt.query().ToolRows().Build().sql(),
      args: mt.query().ToolRows().Build().args(),
    };
    out.queryOne_ok = _try(() => db.queryOne("SELECT COUNT(*) AS n FROM tool_calls"));
    return out;
  } finally {
    db.close();
  }
}

__verb__("utf8", { name: "utf8", short: "UTF-8 byte size", fields: {} });
__verb__("clip", { name: "clip", short: "Permissive clipping at 3 bytes", fields: {} });
__verb__("strict", { name: "strict", short: "Strict rejection at 3 bytes", fields: {} });
__verb__("overflow", { name: "overflow", short: "Row overflow envelope", fields: {} });
__verb__("toolrows", { name: "toolrows", short: "Tool row fidelity fields", fields: {} });
__verb__("tokens", { name: "tokens", short: "Token provenance rows", fields: {} });
__verb__("manifest", { name: "manifest", short: "Corpus identity manifest", fields: {} });
__verb__("envelopes", { name: "envelopes", short: "Exception vs envelope contrast", fields: {} });

function schema() {
  const mt = _mt();
  const db = mt.db().File(_file("archive.json")).SQLiteMemory().Build();
  try {
    const rec = mt.query().TokenUsageRows().Build();
    return { tables: db.tables(), schema: db.schema(), tokenRecipe: { name: rec.name(), sql: rec.sql(), args: rec.args() } };
  } finally {
    db.close();
  }
}
__verb__("schema", { name: "schema", short: "Introspect tables and token recipe", fields: {} });
