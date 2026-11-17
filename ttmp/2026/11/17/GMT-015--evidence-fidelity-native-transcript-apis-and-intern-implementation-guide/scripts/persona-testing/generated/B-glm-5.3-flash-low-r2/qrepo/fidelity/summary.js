// Token/tool summary over explicit archive paths (repeat --files).
__section__("params", {
  fields: {
    files: { type: "stringList", help: "Archive paths" },
  },
});

function summary(params) {
  const mt = require("minitrace");
  const db = mt.db().Files(params.files).SQLiteMemory().Build();
  try {
    const tools = db.query(
      "SELECT session_id, COUNT(*) AS tool_rows, SUM(CASE WHEN outcome_status='succeeded' THEN 1 ELSE 0 END) AS succeeded, SUM(CASE WHEN outcome_status='failed' THEN 1 ELSE 0 END) AS failed, SUM(CASE WHEN outcome_status='unknown' THEN 1 ELSE 0 END) AS unknown_outcome FROM tool_calls GROUP BY session_id");
    const tu = db.execute(mt.query().TokenUsageRows().Build());
    if (tu.error || !tu.complete) throw new Error(tu.error || "incomplete");
    const agg = {};
    for (const r of tu.rows) {
      const k = [r.session_id, r.token_origin, r.token_unit, r.token_method].join("|");
      agg[k] = agg[k] || { session_id: r.session_id, token_origin: r.token_origin, token_unit: r.token_unit, token_method: r.token_method, rows: 0, input_tokens: 0, output_tokens: 0, tool_tokens: 0 };
      agg[k].rows++;
      agg[k].input_tokens += r.input_tokens || 0;
      agg[k].output_tokens += r.output_tokens || 0;
      agg[k].tool_tokens += r.tool_tokens || 0;
    }
    const tokens = Object.values(agg);
    return { tools, tokens };
  } finally { db.close(); }
}

__verb__("summary", {
  name: "summary",
  short: "Token/tool summary for given archives",
  fields: { params: { bind: "params" } },
});
