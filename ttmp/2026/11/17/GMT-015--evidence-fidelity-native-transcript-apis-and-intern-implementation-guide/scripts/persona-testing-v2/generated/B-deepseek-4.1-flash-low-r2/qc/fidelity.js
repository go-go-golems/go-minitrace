__section__("inputs", {
  title: "Inputs",
  fields: {
    archive: { type: "string", help: "archive path" },
    strict_archive: { type: "string", help: "strict archive path" },
    manifest_a: { type: "string", help: "manifest a" },
    manifest_b: { type: "string", help: "manifest b" },
    manifest_c: { type: "string", help: "manifest c" },
  },
});

function fidelity(inputs) {
  const mt = require("minitrace");
  const out = [];
  out.push({metric: "utf8_bytes_abe", value: mt.utf8ByteLength("abé")});

  const db = mt.db().File(inputs.archive).SQLiteMemory()
    .MaxCellBytes(3).Build();
  try {
    const res = db.execute({sql: "SELECT 'abé' AS text"});
    out.push({metric: "permissive_error", value: res.error === undefined ? null : res.error});
    out.push({metric: "permissive_complete", value: res.complete});
    out.push({metric: "permissive_truncated", value: res.truncated});
    out.push({metric: "permissive_count", value: res.count});
    out.push({metric: "permissive_text", value: res.rows.length ? res.rows[0].text : null});
    out.push({metric: "permissive_rows_json", value: JSON.stringify(res.rows)});
    out.push({metric: "permissive_cell_diagnostics_json", value: JSON.stringify(res.cellDiagnostics || null)});
  } finally { db.close(); }

  const sdb = mt.db().File(inputs.strict_archive).SQLiteMemory()
    .MaxCellBytes(3).StrictEvidence(true).Build();
  try {
    const sres = sdb.execute({sql: "SELECT 'abé' AS text"});
    out.push({metric: "strict_error", value: sres.error === undefined ? null : sres.error});
    out.push({metric: "strict_complete", value: sres.complete});
    out.push({metric: "strict_count", value: sres.count});
    out.push({metric: "strict_rows_json", value: JSON.stringify(sres.rows)});
  } finally { sdb.close(); }

  const tdb = mt.db().File(inputs.archive).SQLiteMemory().Build();
  try {
    const r = tdb.execute(mt.query().ToolRows().Build());
    out.push({metric: "toolrows_error", value: r.error === undefined ? null : r.error});
    out.push({metric: "toolrows_rows_json", value: JSON.stringify(r.rows)});
    out.push({metric: "toolrows_columns_json", value: JSON.stringify(r.columns)});
  } finally { tdb.close(); }

  const manifest = mt.corpusManifest([inputs.manifest_a, inputs.manifest_b, inputs.manifest_c]);
  out.push({metric: "manifest_duplicate_count", value: manifest.duplicateCount});
  out.push({metric: "manifest_conflict_count", value: manifest.conflictCount});
  out.push({metric: "manifest_json", value: JSON.stringify(manifest)});
  return out;
}

__verb__("fidelity", {
  name: "fidelity",
  short: "Evidence fidelity checks",
  fields: {
    inputs: { bind: "inputs" }
  }
});
