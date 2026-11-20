// JS/SQL wording checks for UTF-8 byte size, permissive clipping, strict
// rejection and row overflow. No archive data is read here: these are SQL
// literal and limit probes so the byte semantics can be separated from corpus
// loading. Run with e.g.:
//   query commands --query-repository <this repo> unicode unicode byte-size

__section__("opts", {
  title: "Options",
  fields: {
    cell_bytes: { type: "int", default: 3, help: "UTF-8 byte limit per result cell" },
    max_rows: { type: "int", default: 2, help: "Row limit for the overflow probe" },
  },
});

function _memDb(mt, cellBytes, strict, maxRows) {
  return mt.db()
    .SQLiteMemory()
    .MaxCellBytes(cellBytes)
    .StrictEvidence(strict)
    .MaxRows(maxRows)
    .Build();
}

// Exact UTF-8 byte size of "abé": 3 ASCII-ish bytes for a,b plus 2 bytes for é.
function byteSize(opts) {
  const mt = require("minitrace");
  const text = "ab\u00e9";
  const db = _memDb(mt, 4096, false, 100);
  try {
    const rows = db.query("SELECT ? AS text", text);
    return {
      probe: "byte-size",
      text: rows[0].text,
      utf8ByteLength: mt.utf8ByteLength(text),
      // reference count from SQLite itself, character count is 3, byte count 4
      sqlLength: db.query("SELECT length(?) AS chars, length(CAST(? AS BLOB)) AS bytes", text, text)[0],
      matches: rows[0].text === text,
    };
  } finally {
    db.close();
  }
}

// Permissive clipping: 3-byte cell limit, abé must clip to "ab" (never a
// half-written é) and the envelope must report incompleteness.
function permissiveClip(opts) {
  const mt = require("minitrace");
  const db = _memDb(mt, opts.cell_bytes, false, 100);
  try {
    const env = db.queryResult("SELECT 'ab\u00e9' AS text");
    return {
      probe: "permissive-clip",
      cellBytes: opts.cell_bytes,
      text: env.rows[0].text,
      complete: env.complete,
      truncated: env.truncated,
      error: env.error || null,
      count: env.count,
      cellDiagnostics: env.cellDiagnostics,
    };
  } finally {
    db.close();
  }
}

// Strict evidence: same 3-byte limit, no partial rows allowed at all.
function strictClip(opts) {
  const mt = require("minitrace");
  const db = _memDb(mt, opts.cell_bytes, true, 100);
  try {
    const env = db.queryResult("SELECT 'ab\u00e9' AS text");
    return {
      probe: "strict-clip",
      cellBytes: opts.cell_bytes,
      rows: env.rows,
      rowCount: env.rows ? env.rows.length : null,
      count: env.count,
      complete: env.complete,
      truncated: env.truncated,
      error: env.error || null,
      hadError: !!env.error,
      cellDiagnostics: env.cellDiagnostics || null,
    };
  } finally {
    db.close();
  }
}

// Row overflow is a separate axis from cell clipping: cells are well inside
// the byte budget, only the row limit bites.
function rowOverflow(opts) {
  const mt = require("minitrace");
  const db = _memDb(mt, 4096, false, opts.max_rows);
  try {
    const env = db.queryResult(
      "SELECT 'a' AS k UNION ALL SELECT 'b' UNION ALL SELECT 'c'"
    );
    const strictDb = _memDb(mt, 4096, true, opts.max_rows);
    let strictEnv;
    try {
      strictEnv = strictDb.queryResult(
        "SELECT 'a' AS k UNION ALL SELECT 'b' UNION ALL SELECT 'c'"
      );
    } finally {
      strictDb.close();
    }
    return {
      probe: "row-overflow",
      maxRows: opts.max_rows,
      permissive: {
        count: env.count,
        complete: env.complete,
        truncated: env.truncated,
        rowsTruncated: env.rowsTruncated,
        error: env.error || null,
        cellDiagnostics: env.cellDiagnostics || null,
        keys: (env.rows || []).map((r) => r.k),
      },
      strict: {
        count: strictEnv.count,
        complete: strictEnv.complete,
        truncated: strictEnv.truncated,
        rowsTruncated: strictEnv.rowsTruncated,
        error: strictEnv.error || null,
        rowCount: (strictEnv.rows || []).length,
        keys: (strictEnv.rows || []).map((r) => r.k),
      },
    };
  } finally {
    db.close();
  }
}

__verb__("byteSize", {
  name: "byte-size",
  short: "Exact UTF-8 byte size of abé",
  fields: { opts: { bind: "opts" } },
});

__verb__("permissiveClip", {
  name: "permissive-clip",
  short: "Permissive 3-byte cell clipping",
  fields: { opts: { bind: "opts" } },
});

__verb__("strictClip", {
  name: "strict-clip",
  short: "Strict evidence rejection at 3 bytes",
  fields: { opts: { bind: "opts" } },
});

__verb__("rowOverflow", {
  name: "row-overflow",
  short: "Row overflow separate from cell clipping",
  fields: { opts: { bind: "opts" } },
});
