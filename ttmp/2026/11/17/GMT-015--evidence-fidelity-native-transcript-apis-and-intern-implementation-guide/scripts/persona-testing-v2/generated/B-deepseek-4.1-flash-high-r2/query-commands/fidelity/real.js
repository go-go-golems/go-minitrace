const GLOBS = [
  "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/claude-code/active/*/*.minitrace.json",
  "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/codex/active/*/*.minitrace.json",
  "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/pi/active/*/*.minitrace.json",
];

function real() {
  const mt = require("minitrace");
  const out = [];
  for (let i = 0; i < GLOBS.length; i++) {
    const g = GLOBS[i];
    const db = mt.db().Glob(g).SQLiteMemory().Build();
    try {
      const counts = db.execute(mt.query().SessionToolCounts().Build());
      const usage = db.execute(mt.query().TokenUsageRows().Build());
      const summary = db.execute(mt.query().SessionSummary().Build());
      out.push({
        glob: g,
        counts_error: counts.error || null,
        counts: counts.rows || [],
        summary_error: summary.error || null,
        summary: (summary.rows || []).map(function (r) {
          return {
            session_id: r.session_id,
            agent_framework: r.agent_framework,
            model: r.model,
            turn_count: r.turn_count,
            tool_call_count: r.tool_call_count,
            model_invocation_count: r.model_invocation_count,
          };
        }),
        usage_error: usage.error || null,
        usage: usage.rows || [],
      });
    } finally { db.close(); }
  }
  return out;
}

__verb__("real", {
  name: "real",
  short: "Summarize one real archive per framework",
  fields: {},
});
