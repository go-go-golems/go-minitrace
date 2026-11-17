// B-task checks: utf8 bytes, clipping, strict, ordinals, corpusManifest
function b() {
  const mt = require("minitrace");
  const FIX = "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive.json";
  const FIXC = "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive-copy.json";
  const FIXX = "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive-conflict.json";
  const out = {};

  // 1. exact utf8 bytes
  out.utf8_bytes_abe = mt.utf8ByteLength("abé");

  // 2. permissive clipping at 3 bytes
  const dbPermissive = mt.db().RuntimeArchives().SQLiteMemory()
    .MaxCellBytes(3).Build();
  try {
    const r = dbPermissive.execute({sql: "SELECT 'abé' AS text"});
    out.permissive = {
      error: r.error ?? null,
      rows: r.rows,
      count: r.count,
      complete: r.complete,
      truncated: r.truncated ?? null,
      cellDiagnostics: r.cellDiagnostics ?? null,
    };
  } finally { dbPermissive.close(); }

  // 3. strict rejection at 3 bytes
  const dbStrict = mt.db().RuntimeArchives().SQLiteMemory()
    .MaxCellBytes(3).StrictEvidence(true).Build();
  try {
    const r = dbStrict.execute({sql: "SELECT 'abé' AS text"});
    out.strict = {
      error: r.error ?? null,
      rows: r.rows,
      count: r.count,
      complete: r.complete,
    };
  } finally { dbStrict.close(); }

  // 4. ordinals from ToolRows on the fixture archive
  const db = mt.db().RuntimeArchives().SQLiteMemory()
    .MaxCellBytes(16000).StrictEvidence(true).Build();
  try {
    const recipe = mt.query().ToolRows().Build();
    const res = db.execute(recipe);
    if (res.error || !res.complete) throw new Error(res.error || "incomplete");
    const rows = res.rows;
    out.default_toolrows_columns = res.columns;
    out.default_order = rows.map(r => ({
      tool_call_id: r.tool_call_id ?? r.id ?? null,
      source_ordinal: r.source_ordinal,
      timestamp: r.timestamp,
    }));
    out.archive_order = rows.slice().sort((a, b) => a.source_ordinal - b.source_ordinal)
      .map(r => ({tool_call_id: r.tool_call_id ?? r.id ?? null, source_ordinal: r.source_ordinal}));
    out.archive_source_ordinals = rows.map(r => r.source_ordinal)
      .sort((a, b) => a - b);
    out.chronological_source_ordinals = rows.map(r => r.source_ordinal);
  } finally { db.close(); }

  // 5. corpusManifest on exactly the three fixture archives
  const manifest = mt.corpusManifest([FIX, FIXC, FIXX]);
  out.manifest = {
    sessions: manifest.sessions,
    duplicateCount: manifest.duplicateCount,
    conflictCount: manifest.conflictCount,
    entries: manifest.entries.map(e => ({
      path: e.path,
      sessionId: e.sessionId,
      duplicateOf: e.duplicateOf,
      conflict: e.conflict,
      sha256: e.sha256 ?? e.archiveSha256 ?? null,
    })),
  };
  return [out];
}

__verb__("b", {
  name: "b",
  short: "Evidence fidelity checks for task B",
  fields: {},
});
