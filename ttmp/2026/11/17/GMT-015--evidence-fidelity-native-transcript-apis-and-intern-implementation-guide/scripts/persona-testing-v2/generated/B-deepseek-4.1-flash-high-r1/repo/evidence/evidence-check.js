__section__("inputs", {
  title: "Inputs",
  fields: {}
});

function evidenceCheck() {
  const mt = require("minitrace");
  const base = "/private/tmp/gmt015-persona-v2-matrix/fixtures";
  const archive = base + "/archive.json";
  const out = {};

  // Exact UTF-8 byte length, no Buffer/char-count confusion.
  out.utf8_bytes_abe = mt.utf8ByteLength("abé");

  const perm = mt.db().File(archive).SQLiteMemory().MaxCellBytes(3).Build();
  try {
    const r = perm.execute({sql: "SELECT 'abé' AS text"});
    out.perm_error = r.error ?? null;
    out.perm_complete = r.complete ?? null;
    out.perm_count = r.count ?? null;
    out.perm_truncated = r.truncated ?? null;
    out.perm_text = (r.rows && r.rows[0]) ? r.rows[0].text : null;
    out.perm_diag = r.cellDiagnostics ?? null;
    out.perm_diag_len = (r.cellDiagnostics || []).length;

    // Archive tool-array order: explicit ORDER BY source_ordinal.
    const byArchive = perm.execute({
      sql: "SELECT source_ordinal FROM tool_calls ORDER BY source_ordinal"
    });
    out.archive_ordinals = byArchive.rows.map(x => x.source_ordinal);
    out.archive_error = byArchive.error ?? null;

    // DEFAULT ToolRows order (emitting turn, timestamp, tool id).
    const defaultRows = perm.execute(mt.query().ToolRows().Build());
    out.default_error = defaultRows.error ?? null;
    out.default_complete = defaultRows.complete ?? null;
    out.default_ordinals = defaultRows.rows.map(x => x.source_ordinal);
    out.default_tool_ids = defaultRows.rows.map(x => x.tool_call_id);
  } finally {
    perm.close();
  }

  const strict = mt.db().File(archive).SQLiteMemory()
    .MaxCellBytes(3).StrictEvidence(true).Build();
  try {
    const s = strict.execute({sql: "SELECT 'abé' AS text"});
    out.strict_error = s.error ?? null;
    out.strict_error_flag = !!(s.error);
    out.strict_complete = s.complete ?? null;
    out.strict_count = s.count ?? null;
    out.strict_rows = (s.rows || []).length;
    out.strict_columns = (s.columns || []).length;
  } finally {
    strict.close();
  }

  const manifest = mt.corpusManifest([
    base + "/archive.json",
    base + "/archive-copy.json",
    base + "/archive-conflict.json",
  ]);
  out.manifest_duplicate_count = manifest.duplicateCount;
  out.manifest_conflict_count = manifest.conflictCount;
  out.manifest_entries = manifest.entries.map(e => ({
    path: e.path,
    sessionId: e.sessionId,
    sha256: e.sha256,
    duplicateOf: e.duplicateOf ?? null,
    conflict: e.conflict
  }));
  out.manifest_groups = manifest.sessions.map(g => ({
    sessionId: g.sessionId,
    entryIndexes: g.entryIndexes,
    contentVariants: g.contentVariants,
    conflict: g.conflict
  }));
  return [out];
}

__verb__("evidenceCheck", {
  name: "evidence-check",
  short: "Synthetic evidence fidelity checks"
});
