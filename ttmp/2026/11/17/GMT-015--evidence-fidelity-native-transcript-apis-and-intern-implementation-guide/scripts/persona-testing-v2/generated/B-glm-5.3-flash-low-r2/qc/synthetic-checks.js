function syntheticChecks() {
  const mt = require("minitrace");
  const out = {};

  out.utf8_bytes_ab_e = mt.utf8ByteLength("abé");

  // Permissive clipping at 3 bytes
  const perm = mt.db().RuntimeArchives().SQLiteMemory().MaxCellBytes(3).Build();
  try {
    const r = perm.execute({ sql: "SELECT 'abé' AS text" });
    out.clipped_text = r.error ? null : (r.rows.length ? r.rows[0].text : null);
    out.clipped_complete = r.complete === true;
    out.clipped_error = r.error || null;
    out.clipped_diagnostics = r.cellDiagnostics || null;
  } finally { perm.close(); }

  // Strict rejection at 3 bytes
  const strict = mt.db().RuntimeArchives().SQLiteMemory().MaxCellBytes(3).StrictEvidence(true).Build();
  try {
    const r = strict.execute({ sql: "SELECT 'abé' AS text" });
    out.strict_error = !!(r.error);
    out.strict_error_message = r.error || null;
    out.strict_rows = r.rows ? r.rows.length : 0;
    out.strict_count = r.count;
  } finally { strict.close(); }

  // corpusManifest on the three fixture archives
  const manifest = mt.corpusManifest([
    "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive.json",
    "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive-copy.json",
    "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive-conflict.json"
  ]);
  out.manifest_duplicate_count = manifest.duplicateCount;
  out.manifest_conflict_count = manifest.conflictCount;
  out.manifest_entries = manifest.entries.map(e => ({ path: e.path, sessionId: e.sessionId, duplicateOf: e.duplicateOf, conflict: e.conflict }));

  // ToolRows source_ordinal in default (chronological) order and archive order
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try {
    const tr = db.execute(mt.query().ToolRows().Build());
    if (tr.error) { out.toolrows_error = tr.error; }
    const rows = tr.rows || [];
    out.chronological_source_ordinals = rows.map(x => x.source_ordinal);
    out.archive_source_ordinals = rows.map(x => x.source_ordinal).sort((a, b) => a - b);
    out.toolrows_meta = rows.map(x => ({ source_ordinal: x.source_ordinal, tool_call_id: x.tool_call_id, timestamp: x.timestamp }));
  } finally { db.close(); }

  return out;
}

__verb__("syntheticChecks", {
  name: "synthetic-checks",
  short: "Synthetic fixture evidence fidelity checks",
  fields: {}
});
