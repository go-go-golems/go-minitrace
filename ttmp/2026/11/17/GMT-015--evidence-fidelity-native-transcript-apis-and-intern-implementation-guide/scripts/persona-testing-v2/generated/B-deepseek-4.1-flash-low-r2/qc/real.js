__section__("inputs", {
  title: "Inputs",
  fields: { archive: { type: "string", help: "archive path" } },
});

function real(inputs) {
  const mt = require("minitrace");
  const db = mt.db().File(inputs.archive).SQLiteMemory().QueryCommandDefaults().Build();
  try {
    const counts = db.execute(mt.query().SessionToolCounts().Build());
    const usage = db.execute(mt.query().TokenUsageRows().Build());
    const summary = db.execute(mt.query().SessionSummary().Build());
    const tools = db.execute(mt.query().ToolRows().Build());
    const rows = usage.rows || [];
    const recorded = rows.filter(r => r.token_origin === "recorded_usage");
    const unknown = rows.filter(r => r.token_origin === "unknown");
    const sum = (arr, k) => arr.reduce((a, r) => a + (Number(r[k]) || 0), 0);
    const max = (arr, k) => arr.reduce((a, r) => Math.max(a, Number(r[k]) || 0), 0);
    const tokenEvidence = {
      source_recipe: "TokenUsageRows",
      row_count: rows.length,
      recorded_usage_rows: recorded.length,
      unknown_rows: unknown.length,
      token_unit: rows.length ? rows[0].token_unit : null,
      token_methods: [...new Set(rows.map(r => r.token_method))],
      recorded_input_tokens_sum: sum(recorded, "input_tokens"),
      recorded_output_tokens_sum: sum(recorded, "output_tokens"),
      recorded_reasoning_tokens_sum: sum(recorded, "reasoning_tokens"),
      recorded_cache_read_tokens_sum: sum(recorded, "cache_read_tokens"),
      recorded_cache_creation_tokens_sum: sum(recorded, "cache_creation_tokens"),
      recorded_tool_tokens_sum: sum(recorded, "tool_tokens"),
      recorded_total_tokens_sum_over_rows: sum(recorded, "total_tokens"),
      recorded_total_tokens_max_row: max(recorded, "total_tokens"),
      note: "Sums are over observed rows; total_tokens is reported context size, not a re-summed billing total."
    };
    return [
      {metric: "session_id", value: summary.rows.length ? summary.rows[0].session_id : null},
      {metric: "session_tool_counts_json", value: JSON.stringify(counts.rows)},
      {metric: "tool_count", value: counts.rows.length ? counts.rows[0].tool_count : null},
      {metric: "tool_rows_count", value: tools.rows.length},
      {metric: "token_evidence_json", value: JSON.stringify(tokenEvidence)},
      {metric: "session_summary_json", value: JSON.stringify(summary.rows)},
      {metric: "usage_error", value: usage.error === undefined ? null : usage.error},
    ];
  } finally { db.close(); }
}

__verb__("real", { name: "real", short: "Real session evidence", fields: { inputs: { bind: "inputs" } } });
