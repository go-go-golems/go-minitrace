__section__("inputs", { title: "Inputs", fields: {} });
function realProbe() {
  const mt = require("minitrace");
  const files = [
    "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/claude-code/active/2026-10/7dce992c-7793-4e1a-950c-4db92cef2502.minitrace.json",
    "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/codex/active/2026-10/01a13157-f631-7cd3-8f84-e2666b24af01.minitrace.json",
    "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/pi/active/2026-09/01a0a5c3-342f-7447-be74-b6c35f542ff7.minitrace.json"
  ];
  const out = [];
  for (const f of files) {
    const db = mt.db().File(f).SQLiteMemory().Build();
    try {
      const s = db.execute({sql: "SELECT session_id, agent_framework, title, model, turn_count, tool_call_count, total_input_tokens, total_output_tokens, total_cache_read_tokens, total_cache_creation_tokens, total_reasoning_tokens, total_tool_tokens FROM sessions"});
      const tc = db.execute(mt.query().SessionToolCounts().Build());
      const tu = db.execute(mt.query().TokenUsageRows().Build());
      out.push({
        file: f,
        session: s.rows,
        session_error: s.error ?? null,
        tool_counts: tc.rows,
        tool_counts_error: tc.error ?? null,
        token_rows_count: tu.rows.length,
        token_rows_error: tu.error ?? null,
        token_rows_complete: tu.complete ?? null,
        token_rows_sample: tu.rows.slice(0, 3),
        token_columns: tu.columns
      });
    } finally { db.close(); }
  }
  return out;
}
__verb__("realProbe", { name: "real-probe", short: "Probe real archives" });
