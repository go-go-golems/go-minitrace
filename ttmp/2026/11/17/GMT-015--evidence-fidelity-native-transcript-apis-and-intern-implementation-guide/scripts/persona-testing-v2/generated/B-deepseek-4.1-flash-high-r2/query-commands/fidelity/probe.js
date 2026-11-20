const ARCHIVE = "/private/tmp/gmt015-persona-v2-matrix/fixtures/archive.json";

function probe() {
  const mt = require("minitrace");
  const out = { archive: ARCHIVE };

  out.unicode_bytes = mt.utf8ByteLength("ab\u00e9");

  const db1 = mt.db().Archive(ARCHIVE).SQLiteMemory().MaxCellBytes(3).Build();
  try {
    out.permissive = db1.execute({ sql: "SELECT 'ab\u00e9' AS text" });
  } finally { db1.close(); }

  const db2 = mt.db().Archive(ARCHIVE).SQLiteMemory().MaxCellBytes(3).StrictEvidence(true).Build();
  try {
    out.strict = db2.execute({ sql: "SELECT 'ab\u00e9' AS text" });
  } finally { db2.close(); }

  const db3 = mt.db().Archive(ARCHIVE).SQLiteMemory().Build();
  try {
    const r = db3.execute(mt.query().ToolRows().Build());
    out.tool_envelope_error = r.error || null;
    out.tool_complete = r.complete;
    out.tool_rows = (r.rows || []).map(function (row) {
      return {
        tool_call_id: row.tool_call_id,
        source_ordinal: row.source_ordinal,
        timestamp: row.timestamp,
        turn_index: row.turn_index,
      };
    });
  } finally { db3.close(); }

  return out;
}

__verb__("probe", {
  name: "probe",
  short: "Probe synthetic fidelity fixture",
  fields: {},
});
