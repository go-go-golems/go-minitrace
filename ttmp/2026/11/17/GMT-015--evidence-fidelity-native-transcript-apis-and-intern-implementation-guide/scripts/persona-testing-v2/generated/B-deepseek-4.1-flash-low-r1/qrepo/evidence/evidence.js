const FIX = "/private/tmp/gmt015-persona-v2-matrix/fixtures";

function mkDb(paths, maxCellBytes, strict) {
  const mt = require("minitrace");
  let b = mt.db().Sources(mt.sources().Files(paths).Build()).SQLiteMemory();
  if (maxCellBytes) b = b.MaxCellBytes(maxCellBytes);
  if (strict) b = b.StrictEvidence(true);
  return b.Build();
}

function utf8() {
  const mt = require("minitrace");
  return [{ label: "abé", utf8_bytes: mt.utf8ByteLength("abé") }];
}

function permissive() {
  const mt = require("minitrace");
  const db = mkDb([FIX + "/archive.json"], 3, false);
  try {
    return [db.execute({ sql: "SELECT 'abé' AS text" })];
  } finally { db.close(); }
}

function strict() {
  const mt = require("minitrace");
  const db = mkDb([FIX + "/archive.json"], 3, true);
  try {
    return [db.execute({ sql: "SELECT 'abé' AS text" })];
  } finally { db.close(); }
}

function archiveOrder() {
  const mt = require("minitrace");
  const db = mkDb([FIX + "/archive.json"], null, false);
  try {
    return db.query("SELECT source_ordinal, tool_call_id, timestamp, outcome_status FROM tool_calls ORDER BY source_ordinal");
  } finally { db.close(); }
}

function defaultOrder() {
  const mt = require("minitrace");
  const db = mkDb([FIX + "/archive.json"], null, false);
  try {
    const res = db.execute(mt.query().ToolRows().Build());
    if (res.error) throw new Error(res.error);
    return res.rows.map(function (r) { return { source_ordinal: r.source_ordinal, tool_call_id: r.tool_call_id, timestamp: r.timestamp, outcome_status: r.outcome_status }; });
  } finally { db.close(); }
}

function manifest() {
  const mt = require("minitrace");
  return [mt.corpusManifest([FIX + "/archive.json", FIX + "/archive-copy.json", FIX + "/archive-conflict.json"])];
}

function realSession(filters) {
  const mt = require("minitrace");
  const path = filters.path;
  const db = mkDb([path], null, false);
  try {
    const counts = db.query("SELECT session_id, agent_framework, tool_call_count FROM sessions");
    const actual = db.execute(mt.query().SessionToolCounts().Build());
    if (actual.error) throw new Error(actual.error);
    const usage = db.execute(mt.query().TokenUsageRows().Build());
    if (usage.error) throw new Error(usage.error);
    return [{ counts: counts, actual_tool_counts: actual.rows, token_usage_rows: usage.rows }];
  } finally { db.close(); }
}

__verb__("utf8", { name: "utf8", short: "UTF-8 byte length", fields: {} });
__verb__("permissive", { name: "permissive", short: "Permissive clipping", fields: {} });
__verb__("strict", { name: "strict", short: "Strict rejection", fields: {} });
__verb__("archiveOrder", { name: "archive-order", short: "Archive source ordinal order", fields: {} });
__verb__("defaultOrder", { name: "default-order", short: "Default ToolRows order", fields: {} });
__verb__("manifest", { name: "manifest", short: "Corpus manifest", fields: {} });
__section__("filters", { title: "Filters", fields: { path: { type: "string", help: "archive path" } } });
__verb__("realSession", {
  name: "real-session", short: "Real session summary",
  fields: { filters: { bind: "filters" } }
});
