__section__("inputs", { title: "Inputs", fields: {} });
function summarize(mt, f) {
  const db = mt.db().File(f).SQLiteMemory().Build();
  try {
    const s = db.execute({sql: `SELECT session_id, agent_framework, title, model, source_format,
      turn_count, tool_call_count, tool_call_record_count, model_invocation_count, started_at
      FROM sessions`});
    if (s.error) throw new Error("sessions query: " + s.error);
    const session = s.rows[0];
    const sid = session.session_id;

    const tc = db.execute(mt.query().SessionToolCounts().SessionID(sid).Build());
    if (tc.error) throw new Error("tool counts: " + tc.error);
    const rawCount = db.execute({sql: "SELECT COUNT(*) AS n FROM tool_calls WHERE session_id = ?", args: [sid]});
    if (rawCount.error) throw new Error("raw count: " + rawCount.error);

    const tu = db.execute(mt.query().TokenUsageRows().SessionID(sid).Build());
    if (tu.error) throw new Error("token rows: " + tu.error);
    const originCounts = {};
    const methodCounts = {};
    const fieldCounts = {};
    const sum = {input_tokens:0, output_tokens:0, cache_read_tokens:0,
                 cache_creation_tokens:0, reasoning_tokens:0, tool_tokens:0, total_tokens:0};
    let recorded = 0, unknown = 0;
    for (const r of tu.rows) {
      originCounts[r.token_origin] = (originCounts[r.token_origin] || 0) + 1;
      methodCounts[r.token_method] = (methodCounts[r.token_method] || 0) + 1;
      fieldCounts[String(r.reported_usage_fields)] = (fieldCounts[String(r.reported_usage_fields)] || 0) + 1;
      if (r.token_origin === "recorded_usage") {
        recorded++;
        for (const k of Object.keys(sum)) sum[k] += (r[k] || 0);
      } else if (r.token_origin === "unknown") {
        unknown++;
      }
    }

    const m = db.execute({sql: `SELECT total_input_tokens, total_output_tokens,
      total_cache_read_tokens, total_cache_creation_tokens, total_reasoning_tokens,
      total_tool_tokens, turn_count, tool_call_count, tool_call_record_count,
      model_invocation_count FROM metrics WHERE session_id = ?`, args: [sid]});
    const metrics = m.rows[0] || null;

    return {
      session_id: sid,
      agent_framework: session.agent_framework,
      source_format: session.source_format,
      title: session.title,
      model: session.model,
      started_at: session.started_at,
      tool_count: tc.rows[0].tool_count,
      raw_tool_rows: rawCount.rows[0].n,
      session_tool_call_metadata: session.tool_call_count,
      token_evidence: {
        source: "mt.query().TokenUsageRows() + metrics table",
        token_rows: tu.rows.length,
        token_rows_complete: tu.complete,
        token_origin_counts: originCounts,
        token_method_counts: methodCounts,
        reported_usage_fields_counts: fieldCounts,
        token_unit: tu.rows.length ? tu.rows[0].token_unit : null,
        recorded_usage_rows: recorded,
        unknown_origin_rows: unknown,
        recorded_usage_totals: recorded > 0 ? sum : null,
        metrics_totals: metrics ? {
          total_input_tokens: metrics.total_input_tokens ?? null,
          total_output_tokens: metrics.total_output_tokens ?? null,
          total_cache_read_tokens: metrics.total_cache_read_tokens ?? null,
          total_cache_creation_tokens: metrics.total_cache_creation_tokens ?? null,
          total_reasoning_tokens: metrics.total_reasoning_tokens ?? null,
          total_tool_tokens: metrics.total_tool_tokens ?? null
        } : null
      }
    };
  } finally { db.close(); }
}
function realSessions() {
  const mt = require("minitrace");
  return [
    summarize(mt, "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/claude-code/active/2026-10/7dce992c-7793-4e1a-950c-4db92cef2502.minitrace.json"),
    summarize(mt, "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/codex/active/2026-10/01a13157-f631-7cd3-8f84-e2666b24af01.minitrace.json"),
    summarize(mt, "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/pi/active/2026-09/01a0a5c3-342f-7447-be74-b6c35f542ff7.minitrace.json")
  ];
}
__verb__("realSessions", { name: "real-sessions", short: "Summarize one real archive per framework" });
