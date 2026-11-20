__section__("inputs", { title: "Inputs", fields: {} });
function syntheticFinal() {
  const mt = require("minitrace");
  const base = "/private/tmp/gmt015-persona-v2-matrix/fixtures";
  const archive = base + "/archive.json";
  const out = { utf8_byte_length_abe: mt.utf8ByteLength("abé") };

  const clip = mt.db().File(archive).SQLiteMemory().MaxCellBytes(3).Build();
  try {
    const r = clip.execute({sql: "SELECT 'abé' AS text"});
    out.clip = {
      error: r.error ?? null, complete: r.complete, count: r.count,
      text: r.rows[0].text, truncated: r.truncated,
      cellDiagnostics: r.cellDiagnostics
    };
  } finally { clip.close(); }

  const strict = mt.db().File(archive).SQLiteMemory().MaxCellBytes(3).StrictEvidence(true).Build();
  try {
    const s = strict.execute({sql: "SELECT 'abé' AS text"});
    out.strict = {
      error: s.error ?? null, error_present: !!s.error,
      complete: s.complete ?? null, count: s.count ?? null,
      row_count: (s.rows || []).length
    };
  } finally { strict.close(); }

  // No cell limit here: this DB is only for ordering, not clipping.
  const clean = mt.db().File(archive).SQLiteMemory().Build();
  try {
    const a = clean.execute({sql: "SELECT source_ordinal FROM tool_calls ORDER BY source_ordinal"});
    const d = clean.execute(mt.query().ToolRows().Build());
    out.archive_source_ordinals = a.rows.map(r => r.source_ordinal);
    out.default_toolrows = {
      order: d.rows.map(r => r.tool_call_id),
      source_ordinals: d.rows.map(r => r.source_ordinal),
      complete: d.complete, error: d.error ?? null
    };
  } finally { clean.close(); }

  const m = mt.corpusManifest([base + "/archive.json", base + "/archive-copy.json", base + "/archive-conflict.json"]);
  out.manifest = {
    duplicateCount: m.duplicateCount,
    conflictCount: m.conflictCount,
    entries: m.entries.map(e => ({path: e.path, duplicateOf: e.duplicateOf ?? null, conflict: e.conflict, sha256: e.sha256})),
    groups: m.sessions.map(g => ({sessionId: g.sessionId, entryIndexes: g.entryIndexes, contentVariants: g.contentVariants, conflict: g.conflict}))
  };
  return [out];
}
__verb__("syntheticFinal", { name: "synthetic-final", short: "Final synthetic evidence checks" });
