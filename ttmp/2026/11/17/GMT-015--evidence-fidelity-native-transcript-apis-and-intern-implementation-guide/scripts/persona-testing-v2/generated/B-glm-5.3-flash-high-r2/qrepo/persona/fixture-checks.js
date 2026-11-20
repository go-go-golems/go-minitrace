// Fixture checks: utf8 byte length, 3-byte clipping (permissive + strict),
// archive vs default ordering of ToolRows, corpusManifest on 3 fixture paths.
function utf8clip() {
  const mt = require("minitrace");
  const out = {};
  out.utf8_bytes_of_abe_accent = mt.utf8ByteLength("abé");

  const FIX = "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive.json";

  const per = mt.db().File(FIX).SQLiteMemory().MaxCellBytes(3).StrictEvidence(false).Build();
  try {
    out.permissive = per.queryResult("SELECT 'abé' AS text");
  } finally {
    per.close();
  }

  const st = mt.db().File(FIX).SQLiteMemory().MaxCellBytes(3).StrictEvidence(true).Build();
  try {
    out.strict = st.queryResult("SELECT 'abé' AS text");
  } finally {
    st.close();
  }
  return out;
}

function ordinals() {
  const mt = require("minitrace");
  const FIX = "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive.json";
  const db = mt.db().File(FIX).SQLiteMemory().Build();
  try {
    const recipe = mt.query().ToolRows().Build();
    const res = db.execute(recipe);
    if (res.error || !res.complete) {
      return { error: res.error || "incomplete", complete: res.complete };
    }
    const defaultOrder = res.rows.map(r => ({
      tool_call_id: r.tool_call_id,
      source_ordinal: r.source_ordinal,
      turn_index: r.turn_index,
      timestamp: r.timestamp,
    }));
    const archiveOrder = res.rows.slice().sort((a, b) => a.source_ordinal - b.source_ordinal).map(r => ({
      tool_call_id: r.tool_call_id,
      source_ordinal: r.source_ordinal,
    }));
    // Cross-check archive order against direct SQL on tool_calls if possible.
    let sqlOrder = null;
    try {
      sqlOrder = db.query("SELECT tool_call_id, source_ordinal FROM tool_calls ORDER BY source_ordinal");
    } catch (e) {
      sqlOrder = { error: String(e) };
    }
    return {
      complete: res.complete,
      default_order: defaultOrder,
      archive_order: archiveOrder,
      sql_order: sqlOrder,
    };
  } finally {
    db.close();
  }
}

function manifest() {
  const mt = require("minitrace");
  const F = "/private/tmp/gmt015-persona-v2-matrix/fixtures/";
  const m = mt.corpusManifest([F + "archive.json", F + "archive-copy.json", F + "archive-conflict.json"]);
  return {
    duplicateCount: m.duplicateCount,
    conflictCount: m.conflictCount,
    sessions: m.sessions,
    entries: m.entries.map(e => ({
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
  };
}
