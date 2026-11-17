// Evidence fidelity lab for synthetic fixtures.
const mt = require("minitrace");

const FIX = "/private/tmp/gmt015-persona/inputs";
function fidelityLab(filters) {
const results = {};

// 1. Exact UTF-8 byte size of "abé"
results.unicode_bytes = mt.utf8ByteLength("abé");

function makeDB(limits) {
  const b = mt.db().File(FIX + "/archive.json").SQLiteMemory();
  return limits(b).Build();
}

// 2. Permissive clipping at 3 bytes
{
  const db = makeDB((b) => b.MaxCellBytes(3));
  const r = db.queryResult("SELECT 'abé' AS text");
  results.clip = r;
  db.close();
}

// 3. Strict rejection with no partial rows at 3 bytes
{
  const db = makeDB((b) => b.StrictEvidence(true).MaxCellBytes(3));
  const r = db.queryResult("SELECT 'abé' AS text");
  results.strict = r;
  db.close();
}

// 4. Row overflow separately: two rows, MaxRows(1)
{
  const db = makeDB((b) => b.MaxRows(1));
  const r = db.queryResult("SELECT 1 AS n UNION ALL SELECT 2");
  results.rowOverflow = r;
  db.close();
}

// 5. Builder recipe
{
  const db = makeDB((b) => b.MaxCellBytes(16000));
  const recipe = mt.query().ToolRows().Build();
  const r = db.execute(recipe);
  results.builderRecipe = { sql: recipe.sql(), args: recipe.args(), count: r.count, complete: r.complete, error: r.error || null, columns: r.columns };
  results.toolRows = r.rows;
  db.close();
}

// 6. Bound-argument SQL recipe (query with args)
{
  const db = makeDB((b) => b.MaxCellBytes(16000));
  const bound = db.queryResult("SELECT session_id, agent_framework FROM sessions WHERE session_id = ?", ["persona-synthetic"]);
  results.boundSql = bound;
  const one = db.queryOne("SELECT COUNT(*) AS n FROM tool_calls WHERE session_id = ?", ["persona-synthetic"]);
  results.boundOne = one;
  db.close();
}

// 7. Contrast query/queryOne exceptions vs envelopes incl. invalid SQL
{
  const db = makeDB((b) => b.MaxCellBytes(16000));
  try {
    db.query("SELECT * FROM no_such_table");
    results.queryException = null;
  } catch (e) {
    results.queryException = String(e && e.message ? e.message : e);
  }
  try {
    db.queryOne("SELECT * FROM no_such_table");
    results.queryOneException = null;
  } catch (e) {
    results.queryOneException = String(e && e.message ? e.message : e);
  }
  const env = db.queryResult("SELECT * FROM no_such_table");
  results.queryResultEnvelope = { error: env.error, rows: env.rows, count: env.count, complete: env.complete };
  const env2 = db.execute({ sql: "SELECT * FROM no_such_table", args: [] });
  results.executeEnvelope = { error: env2.error, rows: env2.rows, count: env2.count, complete: env2.complete };
  db.close();
}

// 8. source_ordinal / arguments_json / output_status vs outcome_status + ordering
{
  const db = makeDB((b) => b.MaxCellBytes(16000));
  results.archOrder = db.query(`
    SELECT tool_call_id, tool_name, source_ordinal, output_status, outcome_status, arguments_json
    FROM tool_calls WHERE session_id = 'persona-synthetic' ORDER BY source_ordinal`);
  results.chronOrder = db.query(`
    SELECT tool_call_id, tool_name, source_ordinal, timestamp
    FROM tool_calls WHERE session_id = 'persona-synthetic' ORDER BY timestamp`);
  db.close();
}

// 9. Token provenance on fixture
{
  const db = makeDB((b) => b.MaxCellBytes(16000));
  const recipe = mt.query().TokenUsageRows().Build();
  const r = db.execute(recipe);
  results.tokenUsage = r.rows;
  db.close();
}

// 10. corpusManifest on the three fixture archives
{
  const manifest = mt.corpusManifest([FIX + "/archive.json", FIX + "/archive-copy.json", FIX + "/archive-conflict.json"]);
  results.manifest = {
    duplicateCount: manifest.duplicateCount,
    conflictCount: manifest.conflictCount,
    sessions: manifest.sessions,
    entries: manifest.entries.map(e => ({ path: e.path, sessionId: e.sessionId, sha256: e.sha256, duplicateOf: e.duplicateOf, conflict: e.conflict })),
  };
}

  return results;
}

__verb__("fidelityLab", {
  name: "fidelity-lab",
  short: "Evidence fidelity checks on synthetic fixtures",
});
