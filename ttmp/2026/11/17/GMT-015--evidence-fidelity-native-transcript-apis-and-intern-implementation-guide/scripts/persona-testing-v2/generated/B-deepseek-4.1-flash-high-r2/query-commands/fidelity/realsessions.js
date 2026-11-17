const SESSIONS = [
  "7dce992c-7793-4e1a-950c-4db92cef2502",
  "01a1c666-5f02-7b92-984e-99a27afd7a94",
  "01a0a5c3-342f-7447-be74-b6c35f542ff7",
];
const GLOB = "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/*/active/*/*.minitrace.json";

function realsessions() {
  const mt = require("minitrace");
  const db = mt.db().Glob(GLOB).SQLiteMemory().Build();
  try {
    const counts = db.execute(mt.query().SessionToolCounts().Build());
    const usage = db.execute(mt.query().TokenUsageRows().Build());
    const sessions = db.execute({ sql: "SELECT session_id, agent_framework, model, title, turn_count, tool_call_count FROM sessions" });
    const out = [];
    for (let i = 0; i < SESSIONS.length; i++) {
      const sid = SESSIONS[i];
      const srow = (sessions.rows || []).filter(function (r) { return r.session_id === sid; })[0] || null;
      const crow = (counts.rows || []).filter(function (r) { return r.session_id === sid; })[0] || null;
      const rows = (usage.rows || []).filter(function (r) { return r.session_id === sid; });
      const recorded = rows.filter(function (r) { return r.token_origin === "recorded_usage"; });
      const agg = {
        usage_rows: rows.length,
        recorded_usage_rows: recorded.length,
        unknown_origin_rows: rows.filter(function (r) { return r.token_origin === "unknown"; }).length,
        token_unit: rows.length ? rows[0].token_unit : null,
        token_origins: Array.from(new Set(rows.map(function (r) { return r.token_origin; }))),
        token_methods: Array.from(new Set(rows.map(function (r) { return r.token_method; }))),
        scopes: Array.from(new Set(rows.map(function (r) { return r.scope; }))),
        reported_usage_fields_values: Array.from(new Set(rows.map(function (r) { return r.reported_usage_fields; }))).sort(),
        sum_recorded_input_tokens: 0,
        sum_recorded_output_tokens: 0,
        sum_recorded_reasoning_tokens: 0,
        sum_recorded_cache_read_tokens: 0,
        sum_recorded_cache_creation_tokens: 0,
        sum_recorded_total_tokens: 0,
        note: "Sums are row-level TokenUsageRows arithmetic over recorded_usage rows only; not asserted provider billing totals.",
      };
      for (let j = 0; j < recorded.length; j++) {
        const r = recorded[j];
        agg.sum_recorded_input_tokens += r.input_tokens || 0;
        agg.sum_recorded_output_tokens += r.output_tokens || 0;
        agg.sum_recorded_reasoning_tokens += r.reasoning_tokens || 0;
        agg.sum_recorded_cache_read_tokens += r.cache_read_tokens || 0;
        agg.sum_recorded_cache_creation_tokens += r.cache_creation_tokens || 0;
        agg.sum_recorded_total_tokens += r.total_tokens || 0;
      }
      out.push({ session: srow, actual_tool_count: crow ? crow.tool_count : null, token_evidence: agg });
    }
    return out;
  } finally { db.close(); }
}

__verb__("realsessions", {
  name: "realsessions",
  short: "Per-session tool counts and token evidence for selected real archives",
  fields: {},
});
