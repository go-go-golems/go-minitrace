// Summarize one real archive: session identity, actual tool count, observed token evidence.
// Aggregation is done in SQL over normalized tables so no large text cells are emitted
// (avoids lossy clipping making `complete` false and sums unreliable).
__section__("filters", {
  title: "Filters",
  fields: {
    archive_path: {
      type: "string",
      help: "Path to a real .minitrace.json archive",
    },
  },
});

function realSession(filters) {
  const mt = require("minitrace");
  const p = filters.archive_path;
  const db = mt.db().File(p).SQLiteMemory().MaxCellBytes(16000).Build();
  try {
    const sessions = db.query(
      "SELECT session_id, agent_framework, title, turn_count, tool_call_count FROM sessions"
    );

    const countsRes = db.execute(mt.query().SessionToolCounts().Build());

    // Actual tool rows grouped by record kind / outcome, small columns only.
    const toolBreakdown = db.query(
      `SELECT record_kind, outcome_status, COUNT(*) AS n
       FROM tool_calls GROUP BY record_kind, outcome_status ORDER BY record_kind, outcome_status`
    );

    // Recorded token evidence aggregated from the same projection TokenUsageRows uses.
    const usage = mt.query().TokenUsageRows().Build();
    const tokenEvidence = db.queryResult(
      `SELECT token_origin, token_unit, token_method, COUNT(*) AS row_count,
              SUM(input_tokens) AS input_tokens_sum,
              SUM(output_tokens) AS output_tokens_sum,
              SUM(cache_read_tokens) AS cache_read_tokens_sum,
              SUM(cache_creation_tokens) AS cache_creation_tokens_sum,
              SUM(reasoning_tokens) AS reasoning_tokens_sum,
              SUM(tool_tokens) AS tool_tokens_sum,
              SUM(total_tokens) AS total_tokens_sum,
              SUM(reported_usage_fields) AS reported_usage_fields_sum
       FROM (${usage.sql()}) GROUP BY token_origin, token_unit, token_method
       ORDER BY token_origin`, ...usage.args()
    );

    // Session rollups live in the metrics table; select only token columns for cross-checking.
    let rollups = null;
    try {
      rollups = db.query(
        `SELECT session_id, total_input_tokens, total_output_tokens, total_cache_read_tokens,
                total_cache_creation_tokens, total_reasoning_tokens, total_tool_tokens
         FROM metrics`
      );
    } catch (e) {
      rollups = { error: String(e) };
    }

    return {
      archive_path: p,
      sessions,
      session_tool_counts: countsRes.rows,
      counts_error: countsRes.error || null,
      counts_complete: countsRes.complete,
      tool_breakdown: toolBreakdown,
      token_evidence: tokenEvidence.error ? { error: tokenEvidence.error } : tokenEvidence.rows,
      token_evidence_complete: tokenEvidence.complete,
      session_rollups: rollups,
    };
  } finally {
    db.close();
  }
}

__verb__("realSession", {
  name: "real-session",
  short: "Summarize one real archive: identity, tool count, token evidence",
  fields: {
    filters: { bind: "filters" },
  },
});
