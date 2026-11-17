// Evidence-fidelity checks against the synthetic fixtures.
const FIX = "/private/tmp/gmt015-persona/inputs/";

__section__("params", {
  fields: {
    mode: { type: "string", help: "unicode|clipping|strict|overflow|recipe|tools|tokens|manifest" },
  },
});

function fidelity(params) {
  const mt = require("minitrace");
  const out = {};

  const mode = params.mode;

  if (mode === "unicode") {
    out.abe = "abé";
    out.utf8ByteLength = mt.utf8ByteLength("abé");
    return out;
  }

  if (mode === "clipping" || mode === "strict") {
    // Permissive: clip at 3 bytes; strict: reject entirely.
    const strict = mode === "strict";
    const db = mt.db().File(FIX + "archive.json").SQLiteMemory()
      .MaxCellBytes(3).StrictEvidence(strict).Build();
    try {
      const r = db.queryResult("SELECT 'abé' AS text");
      out.envelope = {
        error: r.error || null, complete: r.complete, truncated: !!r.truncated,
        count: r.count, rows: r.rows,
        cellDiagnostics: r.cellDiagnostics || null,
      };
      out.clipped_text = r.rows && r.rows.length ? r.rows[0].text : null;
      out.clipped_complete = r.complete;
    } catch (e) {
      out.threw = String(e);
    } finally { db.close(); }
    return out;
  }

  if (mode === "overflow") {
    const db = mt.db().File(FIX + "archive.json").SQLiteMemory()
      .MaxRows(2).Build();
    try {
      const r = db.queryResult("SELECT value FROM (SELECT 1 AS value UNION ALL SELECT 2 UNION ALL SELECT 3)");
      out.envelope = {
        error: r.error || null, complete: r.complete, truncated: !!r.truncated,
        count: r.count, rowsTruncated: r.rowsTruncated, rows: r.rows,
      };
    } finally { db.close(); }
    return out;
  }

  // Everything below needs the synthetic archive loaded as a source.
  if (mode === "recipe" || mode === "tools" || mode === "tokens") {
    const db = mt.db().File(FIX + "archive.json").SQLiteMemory().Build();
    try {
      if (mode === "recipe") {
        // Builder recipe path
        const recipe = mt.query().ToolRows().Build();
        const r = db.execute(recipe);
        out.recipe = { name: recipe.name(), sql: recipe.sql().slice(0, 200), args: recipe.args() };
        out.recipeEnvelope = { error: r.error || null, complete: r.complete, count: r.count };
        out.recipeRows = r.rows.map(x => ({
          tool_id: x.tool_call_id, source_ordinal: x.source_ordinal,
          output_status: x.output_status, outcome_status: x.outcome_status,
          arguments_json: x.arguments_json, timestamp: x.timestamp,
        }));
        // Bound-argument SQL path
        const b = db.queryResult(
          "SELECT tool_call_id, source_ordinal, output_status, outcome_status FROM tool_calls WHERE tool_call_id = ?",
          ["a-second"]);
        out.boundEnvelope = { error: b.error || null, complete: b.complete, count: b.count };
        out.boundRows = b.rows;
        // Contrast: invalid SQL through queryResult (envelope) vs query (throws)
        const bad = db.queryResult("SELECT no_such_column FROM tool_calls");
        out.invalidViaQueryResult = { error: bad.error || null, complete: bad.complete, count: bad.count, rows: bad.rows };
        try {
          db.query("SELECT no_such_column FROM tool_calls");
          out.invalidViaQueryThrew = null;
        } catch (e) {
          out.invalidViaQueryThrew = String(e).slice(0, 200);
        }
        try {
          db.queryOne("SELECT no_such_column FROM tool_calls");
          out.invalidViaQueryOneThrew = null;
        } catch (e) {
          out.invalidViaQueryOneThrew = String(e).slice(0, 200);
        }
        return out;
      }
      if (mode === "tools") {
        const byArchive = db.query(
          "SELECT tool_call_id, source_ordinal, timestamp FROM tool_calls ORDER BY source_ordinal");
        const byTime = db.query(
          "SELECT tool_call_id, source_ordinal, timestamp FROM tool_calls ORDER BY timestamp, tool_call_id");
        out.archiveOrder = byArchive;
        out.chronologicalOrder = byTime;
        return out;
      }
      if (mode === "tokens") {
        const r = db.queryResult("SELECT * FROM metrics");
        out.metricsEnvelope = { error: r.error || null, complete: r.complete, count: r.count };
        out.metricsRows = r.rows;
        const tu = db.execute(mt.query().TokenUsageRows().Build());
        out.tokenRows = tu.rows;
        return out;
      }
    } finally { db.close(); }
  }

  if (mode === "manifest") {
    const m = mt.corpusManifest([
      FIX + "archive.json",
      FIX + "archive-copy.json",
      FIX + "archive-conflict.json",
    ]);
    return { manifest: m };
  }

  throw new Error("unknown mode: " + mode);
}

__verb__("fidelity", {
  name: "fidelity",
  short: "Evidence-fidelity checks on synthetic fixtures",
  fields: { params: { bind: "params" } },
});
