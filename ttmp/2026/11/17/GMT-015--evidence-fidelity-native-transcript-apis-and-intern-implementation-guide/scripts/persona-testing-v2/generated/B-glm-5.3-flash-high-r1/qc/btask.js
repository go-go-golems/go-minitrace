function btask() {
  const mt = require("minitrace");
  const FIX = "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive.json";
  const out = [];

  // 1. Exact UTF-8 byte length of "abé"
  out.push({ kind: "utf8_bytes_abé", value: mt.utf8ByteLength("abé") });

  // 2. Permissive clipping at 3 bytes per cell
  const pdb = mt.db().File(FIX).SQLiteMemory().MaxCellBytes(3).Build();
  try {
    const pres = pdb.queryResult("SELECT 'abé' AS text");
    out.push({ kind: "clip_text", value: pres.rows.length ? String(pres.rows[0].text) : null });
    out.push({ kind: "clip_complete", value: pres.complete === true });
    if (pres.cellDiagnostics && pres.cellDiagnostics.length) {
      const d = pres.cellDiagnostics[0];
      out.push({ kind: "clip_diag", value: JSON.stringify({ originalBytes: d.originalBytes, emittedBytes: d.emittedBytes, truncated: d.truncated }) });
    } else {
      out.push({ kind: "clip_diag", value: null });
    }
  } finally { pdb.close(); }

  // 3. Strict rejection at 3 bytes per cell
  const sdb = mt.db().File(FIX).SQLiteMemory().MaxCellBytes(3).StrictEvidence(true).Build();
  try {
    const sres = sdb.queryResult("SELECT 'abé' AS text");
    out.push({ kind: "strict_error", value: sres.error ? true : false });
    out.push({ kind: "strict_rows", value: Array.isArray(sres.rows) ? sres.rows.length : -1 });
    if (sres.error) out.push({ kind: "strict_error_msg", value: String(sres.error) });
  } finally { sdb.close(); }

  // 4. ToolRows: default order vs archive (source_ordinal) order
  const tdb = mt.db().File(FIX).SQLiteMemory().Build();
  try {
    const tres = tdb.execute(mt.query().ToolRows().Build());
    if (tres.error || !tres.complete) {
      out.push({ kind: "toolrows_error", value: String(tres.error || "incomplete") });
    } else {
      out.push({ kind: "chronological_source_ordinals", value: JSON.stringify(tres.rows.map(r => r.source_ordinal)) });
      const sorted = tdb.query("SELECT source_ordinal FROM tool_calls ORDER BY source_ordinal");
      out.push({ kind: "archive_source_ordinals", value: JSON.stringify(sorted.map(r => r.source_ordinal)) });
      out.push({ kind: "toolrows_detail", value: JSON.stringify(tres.rows.map(r => ({ id: r.tool_call_id, ts: r.timestamp, ord: r.source_ordinal }))) });
    }
  } finally { tdb.close(); }

  // 5. corpusManifest over the three fixture paths
  const manifest = mt.corpusManifest([
    "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive.json",
    "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive-copy.json",
    "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive-conflict.json",
  ]);
  out.push({ kind: "manifest_duplicate_count", value: manifest.duplicateCount });
  out.push({ kind: "manifest_conflict_count", value: manifest.conflictCount });
  out.push({ kind: "manifest_entries", value: JSON.stringify(manifest.entries.map(e => ({ path: e.path.split("/").pop(), sessionId: e.sessionId, duplicateOf: e.duplicateOf, conflict: e.conflict }))) });

  return out;
}

__verb__("btask", {
  name: "btask",
  short: "Synthetic evidence-fidelity checks",
  fields: {},
});
