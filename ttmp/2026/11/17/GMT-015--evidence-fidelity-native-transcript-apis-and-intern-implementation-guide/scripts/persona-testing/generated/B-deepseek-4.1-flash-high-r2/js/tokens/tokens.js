// Token provenance: missing usage vs explicitly recorded zero usage vs
// heuristic estimates, kept in separate units and never summed together.
//
// Key semantics observed in the normalized schema:
//   * turns.<field> is NULL when the archive omitted usage; the turn-level
//     TokenUsageRows() recipe COALESCEs it to 0, so a 0 in the projection is
//     NOT proof the provider reported zero.
//   * token_origin 'unknown' / reported_usage_fields 0  => missing evidence
//   * token_origin 'recorded_usage' with reported_usage_fields > 0 => at least
//     one field was recorded (2 here), so the zeros are real zeros
//   * tool rows use token_unit 'estimated_tokens' + method
//     round_utf8_bytes_div_4 and only appear with .ByTool()

__section__("opts", {
  title: "Options",
  fields: {
    session_id: { type: "string", default: "", help: "Optional session id filter" },
    limit: { type: "int", default: 10, help: "Max sessions in the framework summary" },
  },
});

function _db(mt) {
  return mt.db().RuntimeArchives().SQLiteMemory().MaxCellBytes(8192).StrictEvidence(false).Build();
}

function provenance(opts) {
  const mt = require("minitrace");
  const db = _db(mt);
  try {
    const sid = opts.session_id;
    const turnRows = db.execute(mt.query().TokenUsageRows().SessionID(sid).Build()).rows;
    const toolRows = db.execute(mt.query().TokenUsageRows().SessionID(sid).ByTool().Build()).rows;
    const raw = db.query(
      "SELECT turn_index, role, input_tokens, output_tokens, " +
      "input_tokens IS NULL AS input_is_null, output_tokens IS NULL AS output_is_null, " +
      "json_extract(raw_json, '$.usage') IS NULL AS raw_usage_missing " +
      "FROM turns WHERE (? = '' OR session_id = ?) ORDER BY session_id, turn_index",
      sid, sid
    );
    // Deliberately wrong aggregation, to show what "sum totals" hides:
    const naive = db.query(
      "SELECT SUM(COALESCE(input_tokens,0)+COALESCE(output_tokens,0)) AS naive_total, " +
      "SUM((input_tokens IS NOT NULL) OR (output_tokens IS NOT NULL)) AS turns_with_usage, " +
      "COUNT(*) AS turns FROM turns WHERE (? = '' OR session_id = ?)",
      sid, sid
    );
    return {
      probe: "token-provenance",
      turnScope: turnRows,
      toolScopeEstimates: toolRows,
      rawNullability: raw,
      naiveAggregate: naive[0],
      units: {
        recorded: "tokens (provider-reported fields, zero-filled when absent)",
        estimate: "estimated_tokens (round_utf8_bytes_div_4 heuristic)",
        rule: "never add estimated_tokens to recorded tokens; never read a zero-filled 0 as reported zero",
      },
    };
  } finally {
    db.close();
  }
}

// Per-framework recorded-token and tool summary for the archives in
// --archive-glob. Two independent queries keep the units apart:
//   * recorded tokens come only from turns (token_unit 'tokens')
//   * estimated tool-output tokens come only from tool_calls
function frameworkSummary(opts) {
  const mt = require("minitrace");
  const db = _db(mt);
  try {
    const perSession = db.query(
      "SELECT s.session_id, s.agent_framework, s.turn_count, s.tool_call_count, " +
      "(SELECT COUNT(*) FROM turns t WHERE t.session_id = s.session_id) AS actual_turns, " +
      "(SELECT COUNT(*) FROM tool_calls c WHERE c.session_id = s.session_id) AS actual_tools " +
      "FROM sessions s ORDER BY s.agent_framework, s.session_id LIMIT ?",
      opts.limit
    );
    const recorded = db.query(
      "SELECT s.agent_framework, " +
      "COUNT(*) AS turn_rows, " +
      "SUM((t.input_tokens IS NOT NULL) OR (t.output_tokens IS NOT NULL) OR " +
      "    (t.cache_read_tokens IS NOT NULL) OR (t.cache_creation_tokens IS NOT NULL) OR " +
      "    (t.reasoning_tokens IS NOT NULL) OR (t.tool_tokens IS NOT NULL)) AS turns_any_recorded, " +
      "SUM(COALESCE(t.input_tokens,0)) AS input_tokens_zero_filled, " +
      "SUM(COALESCE(t.output_tokens,0)) AS output_tokens_zero_filled " +
      "FROM turns t JOIN sessions s ON s.session_id = t.session_id " +
      "GROUP BY s.agent_framework ORDER BY s.agent_framework"
    );
    const toolOutcomes = db.query(
      "SELECT s.agent_framework, c.record_kind, c.outcome_status, c.output_status, COUNT(*) AS n " +
      "FROM tool_calls c JOIN sessions s ON s.session_id = c.session_id " +
      "GROUP BY s.agent_framework, c.record_kind, c.outcome_status, c.output_status " +
      "ORDER BY s.agent_framework, n DESC"
    );
    const estimates = db.execute(mt.query().TokenUsageRows().ByTool().Build()).rows || [];
    const estimatedByFramework = {};
    for (const r of estimates) {
      const key = r.session_id;
      if (!estimatedByFramework[key]) {
        estimatedByFramework[key] = { estimated_tokens: 0, rows: 0, units: new Set() };
      }
      estimatedByFramework[key].estimated_tokens += Number(r.total_tokens || 0);
      estimatedByFramework[key].rows += 1;
      estimatedByFramework[key].units.add(r.token_unit);
    }
    const estimateSummary = Object.keys(estimatedByFramework).map((k) => ({
      session_id: k,
      estimated_tokens: estimatedByFramework[k].estimated_tokens,
      estimated_rows: estimatedByFramework[k].rows,
      units: Array.from(estimatedByFramework[k].units),
    }));
    return {
      probe: "framework-summary",
      sessions: perSession,
      recordedUsage: recorded,
      toolOutcomeStatus: toolOutcomes,
      estimateSummary: estimateSummary,
      unitsNote: "recordedUsage rows are token_unit 'tokens'; estimateSummary rows are token_unit 'estimated_tokens'",
    };
  } finally {
    db.close();
  }
}

__verb__("provenance", {
  name: "provenance",
  short: "Missing vs explicit-zero usage and tool estimates",
  fields: { opts: { bind: "opts" } },
});
__verb__("frameworkSummary", {
  name: "framework-summary",
  short: "Recorded tokens and tool outcomes per framework",
  fields: { opts: { bind: "opts" } },
});
