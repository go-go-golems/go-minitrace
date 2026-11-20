// Evidence-fidelity probe verbs for GMT-015 persona matrix.
// Run with: go-minitrace query commands --query-repository <this dir> evidence fidelity <verb>

const FIXTURES = "/private/tmp/gmt015-persona/inputs";
const FIX_DB = FIXTURES + "/archive.json";

function envelopeSummary(r) {
  return {
    columns: r.columns,
    count: r.count,
    complete: r.complete,
    truncated: r.truncated === true,
    rowsTruncated: r.rowsTruncated === undefined ? null : r.rowsTruncated,
    error: r.error === undefined ? null : r.error,
    rows: r.rows,
    cellDiagnostics: r.cellDiagnostics === undefined ? null : r.cellDiagnostics,
  };
}

// ---- 1. Exact UTF-8 byte size + permissive clipping at 3 bytes + strict rejection ----
function unicodeFidelity() {
  const mt = require("minitrace");
  const bytes = mt.utf8ByteLength("abé");
  const out = { utf8ByteLength_abé: bytes };

  // permissive clipping
  const permissive = mt.db().RuntimeArchives().SQLiteMemory().MaxCellBytes(3).Build();
  try {
    out.permissive = envelopeSummary(
      permissive.queryResult("SELECT 'abé' AS text")
    );
  } finally { permissive.close(); }

  // strict rejection
  const strict = mt.db().RuntimeArchives().SQLiteMemory().MaxCellBytes(3).StrictEvidence(true).Build();
  try {
    out.strict = envelopeSummary(
      strict.queryResult("SELECT 'abé' AS text")
    );
  } finally { strict.close(); }

  return out;
}

// ---- 2. Row overflow, separately from cell clipping ----
function rowOverflow() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().MaxRows(2).Build();
  try {
    return envelopeSummary(db.queryResult(
      "SELECT 1 AS n UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4"
    ));
  } finally { db.close(); }
}

// ---- 3. Builder recipe + bound-argument SQL; query/queryOne exceptions vs envelopes ----
function apiEnvelopes() {
  const mt = require("minitrace");
  const out = {};
  const db = mt.db().Archive(FIX_DB).SQLiteMemory().Build();
  try {
    // bound-argument SQL
    out.boundQuery = db.query("SELECT ? AS a, ? AS b", 7, "abé");
    out.boundQueryOne = db.queryOne("SELECT ? AS n", 42);

    // builder recipe through execute()
    const recipe = mt.query().ToolRows().Build();
    const r = db.execute(recipe);
    out.recipeExecute = {
      name: recipe.name(),
      sql: recipe.sql(),
      complete: r.complete,
      error: r.error === undefined ? null : r.error,
      count: r.count,
      rows: r.rows,
    };

    // invalid SQL: query() throws
    try {
      db.query("SELEC 1 AS x");
      out.queryInvalid = "NO THROW (unexpected)";
    } catch (e) {
      out.queryInvalid = { threw: true, message: String(e && e.message ? e.message : e) };
    }

    // invalid SQL: queryOne() throws too
    try {
      db.queryOne("SELEC 1 AS x");
      out.queryOneInvalid = "NO THROW (unexpected)";
    } catch (e) {
      out.queryOneInvalid = { threw: true, message: String(e && e.message ? e.message : e) };
    }

    // invalid SQL: queryResult() returns an error envelope
    out.queryResultInvalid = envelopeSummary(db.queryResult("SELEC 1 AS x"));

    // invalid SQL through execute() with a raw {sql, args} object
    out.executeInvalid = envelopeSummary(db.execute({ sql: "SELEC 1 AS x", args: [] }));

    // empty result through queryOne (no exception?)
    out.queryOneEmpty = db.queryOne("SELECT 1 AS x WHERE 0");
  } finally { db.close(); }
  return out;
}

// ---- 4. Tool source_ordinal / arguments_json / output_status vs outcome_status ----
function toolOrder() {
  const mt = require("minitrace");
  const db = mt.db().Archive(FIX_DB).SQLiteMemory().Build();
  const cols = `tool_name, source_ordinal, timestamp, arguments_json,
                output_status, outcome_status, success`;
  try {
    return {
      archiveOrder: db.query(
        `SELECT ${cols} FROM tool_calls ORDER BY session_id, source_ordinal`),
      chronologicalOrder: db.query(
        `SELECT ${cols} FROM tool_calls ORDER BY timestamp, source_ordinal`),
    };
  } finally { db.close(); }
}

// ---- 5. Token provenance: missing vs explicit zero, estimates, units ----
function tokenProvenance() {
  const mt = require("minitrace");
  const db = mt.db().Archive(FIX_DB).SQLiteMemory().Build();
  try {
    const usage = db.execute(mt.query().TokenUsageRows().Build());
    return {
      complete: usage.complete,
      error: usage.error === undefined ? null : usage.error,
      rows: usage.rows,
    };
  } finally { db.close(); }
}

// ---- 5b. Tool-output estimates (separate origin/method) ----
function tokenEstimates() {
  const mt = require("minitrace");
  const db = mt.db().Archive(FIX_DB).SQLiteMemory().Build();
  try {
    const usage = db.execute(mt.query().TokenUsageRows().ByTool().Build());
    return {
      complete: usage.complete,
      error: usage.error === undefined ? null : usage.error,
      rows: usage.rows,
    };
  } finally { db.close(); }
}

// ---- 6. corpusManifest over the three corpus-identity fixtures ----
function manifest() {
  const mt = require("minitrace");
  const m = mt.corpusManifest([
    FIXTURES + "/archive.json",
    FIXTURES + "/archive-copy.json",
    FIXTURES + "/archive-conflict.json",
  ]);
  return {
    duplicateCount: m.duplicateCount,
    conflictCount: m.conflictCount,
    sessions: m.sessions,
    entries: m.entries.map(e => ({
      index: e.index,
      path: e.path,
      sessionId: e.sessionId,
      originalSessionId: e.originalSessionId,
      sourceFingerprint: e.sourceFingerprint,
      sourceFormat: e.sourceFormat,
      sha256: e.sha256,
      sizeBytes: e.sizeBytes,
      duplicateOf: e.duplicateOf,
      conflict: e.conflict,
    })),
    groups: m.groups,
  };
}

// ---- 7. Token/tool summary over real archives (one per framework) ----
__section__("filters", {
  fields: {
    framework: { type: "stringList", help: "Filter by framework" },
  },
});

function tokenToolSummary(filters) {
  const mt = require("minitrace");
  const fw = filters.framework && filters.framework.length
    ? filters.framework : null;
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try {
    const where = fw ? `WHERE agent_framework IN (${mt.sql.stringIn(fw)})` : "";
    const tools = db.query(`
      SELECT agent_framework, COUNT(*) AS tool_calls,
             SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) AS success_true,
             SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) AS success_false,
             SUM(CASE WHEN success IS NULL THEN 1 ELSE 0 END) AS success_unknown
      FROM tool_calls t JOIN sessions s USING (session_id)
      ${where}
      GROUP BY agent_framework`);
    const usage = db.execute(mt.query().TokenUsageRows().Build());
    if (usage.error) throw new Error(usage.error);
    const fwBySession = {};
    for (const s of db.query("SELECT session_id, agent_framework FROM sessions")) {
      fwBySession[s.session_id] = s.agent_framework;
    }
    const byKey = {};
    for (const row of usage.rows) {
      const framework = fwBySession[row.session_id];
      if (fw && !fw.includes(framework)) continue;
      const key = [framework, row.token_origin, row.token_unit, row.token_method].join("|");
      const bucket = byKey[key] || {
        agent_framework: framework,
        token_origin: row.token_origin,
        token_unit: row.token_unit,
        token_method: row.token_method,
        rows: 0,
        input_tokens: 0,
        output_tokens: 0,
        estimated_tokens: 0,
      };
      bucket.rows += 1;
      const n = (v) => (typeof v === "number" ? v : 0);
      bucket.input_tokens += n(row.input_tokens);
      bucket.output_tokens += n(row.output_tokens);
      bucket.estimated_tokens += n(row.estimated_tokens);
      byKey[key] = bucket;
    }
    return {
      toolSummary: tools,
      tokenSummary: Object.values(byKey),
      tokenRowCount: usage.rows.length,
    };
  } finally { db.close(); }
}

__verb__("unicodeFidelity", { name: "unicode-fidelity", short: "UTF-8 byte size, 3-byte clipping, strict rejection" });
__verb__("rowOverflow", { name: "row-overflow", short: "Row overflow at MaxRows(2)" });
__verb__("apiEnvelopes", { name: "api-envelopes", short: "Builder recipe + bound args + envelope vs exception" });
__verb__("toolOrder", { name: "tool-order", short: "Archive order vs chronological order via source_ordinal" });
__verb__("tokenProvenance", { name: "token-provenance", short: "Token origin/unit/method and estimates" });
__verb__("tokenEstimates", { name: "token-estimates", short: "Tool-output token estimates via ByTool" });
__verb__("manifest", { name: "manifest", short: "corpusManifest over identity fixtures" });
__verb__("tokenToolSummary", { name: "token-tool-summary", short: "Per-framework token/tool summary", fields: { filters: { bind: "filters" } } });
