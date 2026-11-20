// Run in the minitrace JS runtime. Choose paths explicitly before opening a corpus.
const mt = require("minitrace");
const manifest = mt.corpusManifest([]); // replace with normalized archive paths
if (manifest.conflictCount) throw new Error("Resolve same-ID archive variants before opening the corpus");
// ToolRows preserves chronological default; archive order is an explicit query:
const recipe = mt.query().ToolRows().SessionID("session-id").Build();
const archiveOrderSQL = "SELECT * FROM (" + recipe.sql() + ") ORDER BY session_id, source_ordinal";
// Execute using db.queryResult(archiveOrderSQL, ...recipe.args()) and check fidelity.
// TokenUsageRows returns token_origin, token_unit and token_method: never mix
// estimated_tokens with recorded usage when reporting provider accounting.
({ manifest, archiveOrderSQL });
