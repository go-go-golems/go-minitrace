// Token/tool provenance summary for real normalized archives.
// go-minitrace query commands --query-repository ./qrepo probe corpus summary \
//   --glob '/path/*/active/*/*.minitrace.json' --label pi --output json
__section__("filters", {
  title: "Filters",
  fields: {
    glob: { type: "string", help: "Archive glob (one framework)" },
    label: { type: "string", default: "corpus", help: "Label for this run" },
  },
});

function summary(filters) {
  const mt = require("minitrace");
  const db = mt.db()
    .Glob(filters.glob)
    .SQLiteMemory()
    .MaxRows(1000)
    .MaxCellBytes(4000)
    .Build();
  try {
    const turnRecipe = mt.query().TokenUsageRows().Build();
    const toolRecipe = mt.query().TokenUsageRows().ByTool().Build();
    const sessions = db.query(`
      SELECT session_id, agent_framework, model, turn_count, tool_call_count
      FROM sessions ORDER BY session_id`);
    const frameworks = db.query(`
      SELECT agent_framework AS framework, COUNT(*) AS sessions,
             SUM(turn_count) AS turns, SUM(tool_call_count) AS tool_records
      FROM sessions GROUP BY 1 ORDER BY 1`);
    // Recorded turn-token provenance (zero-filled columns; reported_usage_fields
    // is the only in-recipe signal for "was any usage field present").
    const tokenProvenance = db.query(`
      SELECT token_origin, token_unit, token_method, COUNT(*) AS rows,
             SUM(input_tokens) AS input_tokens, SUM(output_tokens) AS output_tokens,
             SUM(total_tokens) AS total_tokens, SUM(reported_usage_fields) AS reported_fields
      FROM (${turnRecipe.sql()})
      GROUP BY 1,2,3 ORDER BY rows DESC`, ...turnRecipe.args());
    // Tool-output estimates: a DIFFERENT unit; never add to the above.
    const toolEstimates = db.query(`
      SELECT token_origin, token_unit, token_method, COUNT(*) AS rows,
             SUM(total_tokens) AS total_tokens
      FROM (${toolRecipe.sql()})
      GROUP BY 1,2,3 ORDER BY rows DESC`, ...toolRecipe.args());
    const outcomes = db.query(`
      SELECT output_status, outcome_status, success, COUNT(*) AS rows
      FROM tool_calls GROUP BY 1,2,3 ORDER BY rows DESC`);
    const ordinals = db.query(`
      SELECT session_id, COUNT(*) AS tools,
             MIN(source_ordinal) AS min_ord, MAX(source_ordinal) AS max_ord,
             COUNT(DISTINCT source_ordinal) AS distinct_ord
      FROM tool_calls GROUP BY 1 ORDER BY 1`);
    return {
      label: filters.label,
      glob: filters.glob,
      cache: db.cacheInfo(),
      sources: String(db.sources()),
      frameworks, sessions, tokenProvenance, toolEstimates, outcomes, ordinals,
      note: "recorded tokens (unit 'tokens') and estimates (unit 'estimated_tokens') are not summed together",
    };
  } finally {
    db.close();
  }
}

__verb__("summary", {
  name: "summary",
  short: "Token/tool provenance summary across archives",
  fields: { filters: { bind: "filters" } },
});
