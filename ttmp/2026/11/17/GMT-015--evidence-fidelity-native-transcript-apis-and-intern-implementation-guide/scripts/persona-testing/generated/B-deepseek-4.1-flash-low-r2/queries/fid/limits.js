__section__("filters", {
  title: "Filters",
  fields: {
    text: { type: "string", default: "abé" },
    limit: { type: "int", default: 10 }
  }
});

function unicodeBytes(filters) {
  filters = filters || {};
  const mt = require("minitrace");
  const text = filters.text || "abé";
  return [{
    text: text,
    utf8_byte_length: mt.utf8ByteLength(text),
    js_string_length: text.length
  }];
}

function clipByte3(filters) {
  filters = filters || {};
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory()
    .MaxCellBytes(3).Build();
  try {
    const text = filters.text || "abé";
    const result = db.queryResult("SELECT ? AS text", text);
    return [{
      input_original_bytes: mt.utf8ByteLength(text),
      rows: result.rows,
      count: result.count,
      complete: result.complete,
      truncated: result.truncated,
      cellDiagnostics: result.cellDiagnostics,
      error: result.error || null
    }];
  } finally {
    db.close();
  }
}

function strictByte3(filters) {
  filters = filters || {};
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory()
    .MaxCellBytes(3).StrictEvidence(true).Build();
  try {
    const result = db.queryResult("SELECT ? AS text", filters.text || "abé");
    return [{
      rows: result.rows || [],
      row_count: (result.rows || []).length,
      count: result.count,
      complete: result.complete,
      truncated: result.truncated,
      error: result.error || null,
      envelope_keys: Object.keys(result)
    }];
  } finally {
    db.close();
  }
}

function rowOverflow(filters) {
  filters = filters || {};
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory()
    .MaxRows(2).MaxCellBytes(4000).Build();
  try {
    const result = db.queryResult(
      "SELECT tool_call_id FROM tool_calls ORDER BY source_ordinal");
    return [{
      rows: result.rows,
      count: result.count,
      complete: result.complete,
      truncated: result.truncated,
      rowsTruncated: result.rowsTruncated,
      cellDiagnostics: result.cellDiagnostics,
      error: result.error || null
    }];
  } finally {
    db.close();
  }
}

__verb__("unicodeBytes", {
  name: "unicode-bytes",
  short: "Exact UTF-8 byte length",
  fields: { filters: { bind: "filters" } }
});

__verb__("clipByte3", {
  name: "clip-byte3",
  short: "Permissive 3-byte cell clipping",
  fields: { filters: { bind: "filters" } }
});

__verb__("strictByte3", {
  name: "strict-byte3",
  short: "Strict evidence 3-byte cell rejection",
  fields: { filters: { bind: "filters" } }
});

__verb__("rowOverflow", {
  name: "row-overflow",
  short: "Row overflow envelope",
  fields: { filters: { bind: "filters" } }
});
