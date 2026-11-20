__verb__("utf8Demo", {
  name: "utf8-demo",
  short: "Demonstrate utf8ByteLength, 3-byte cell limit, and strict evidence rejection",
});

// Tiny synthetic archive: two turns, first one contains "abé" (4 UTF-8 bytes).
const ARCHIVE = {
  id: "utf8-demo-0001",
  schema_version: "minitrace-v0.2.0",
  profile: "organic",
  title: "utf8 demo session abé",
  classification: "internal",
  flags: { for_research: false, needs_cleaning: false, contains_error: false, contains_pii: false, category: [] },
  environment: { model: "demo-model", agent_framework: "demo", platform_type: "agent" },
  timing: { started_at: "2026-01-01T00:00:00Z", ended_at: "2026-01-01T00:01:00Z" },
  turns: [
    { index: 0, timestamp: "2026-01-01T00:00:00Z", role: "user", source: "human", input_channel: "user_input", content: "abé" },
    { index: 1, timestamp: "2026-01-01T00:00:10Z", role: "assistant", source: "agent", input_channel: "agent_output", content: "réponse abé enough to clip" },
  ],
  tool_calls: [],
  annotations: [],
  metrics: { turn_count: 2, tool_call_count: 0, read_count: 0, modify_count: 0, create_count: 0, execute_count: 0, delegate_count: 0, read_ratio: 0 },
};

function utf8Demo() {
  const mt = require("minitrace");
  const out = {};

  // 1. Exact UTF-8 byte length of "abé" (a=1, b=1, é=2 bytes -> 4)
  out.utf8ByteLength_abe = mt.utf8ByteLength("abé");

  // 2. Query the synthetic archive with a 3-byte cell limit.
  //    "abé" is 4 UTF-8 bytes, so a 3-byte limit must clip it to "ab".
  const sources = mt.sources().Content(JSON.stringify(ARCHIVE)).Name("utf8-demo-0001.minitrace.json").Build();
  const db = mt.db()
    .Sources(sources)
    .SQLiteMemory()
    .MaxCellBytes(3)
    .Build();
  try {
    out.lenientResult = db.queryResult("SELECT content AS text FROM turns WHERE turn_index = 0");
    // 3. Same query with StrictEvidence(true): must return an error envelope
    //    with empty rows and count 0.
    const strictSources = mt.sources().Content(JSON.stringify(ARCHIVE)).Name("utf8-demo-0001.minitrace.json").Build();
    const strictDb = mt.db()
      .Sources(strictSources)
      .SQLiteMemory()
      .MaxCellBytes(3)
      .StrictEvidence(true)
      .Build();
    try {
      out.strictResult = strictDb.queryResult("SELECT content AS text FROM turns WHERE turn_index = 0");
    } finally {
      strictDb.close();
    }
  } finally {
    db.close();
  }
  return out;
}
