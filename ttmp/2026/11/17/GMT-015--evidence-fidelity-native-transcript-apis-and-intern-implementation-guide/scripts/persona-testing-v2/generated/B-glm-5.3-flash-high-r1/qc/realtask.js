function realtask() {
  const mt = require("minitrace");
  const out = [];
  const picks = [
    ["claude-code", "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/claude-code/active/2026-10/7dce992c-7793-4e1a-950c-4db92cef2502.minitrace.json"],
    ["codex", "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/codex/active/2026-10/01a13157-f631-7cd3-8f84-e2666b24af01.minitrace.json"],
    ["pi", "/private/tmp/gmt015-persona-v2-matrix/corpus/archives/pi/active/2026-09/01a0a5c3-342f-7447-be74-b6c35f542ff7.minitrace.json"],
  ];
  for (const [fw, path] of picks) {
    const db = mt.db().File(path).SQLiteMemory().Build();
    try {
      const sess = db.queryOne("SELECT session_id, agent_framework, title, turn_count, tool_call_count FROM sessions");
      const counts = db.execute(mt.query().SessionToolCounts().Build());
      const toolCount = (counts.error || !counts.complete) ? null : counts.rows.reduce((a, r) => a + r.tool_count, 0);
      const tok = db.execute(mt.query().TokenUsageRows().Build());
      let tokenEvidence = null;
      if (!tok.error && tok.complete) {
        const byOrigin = {}, byMethod = {}, fieldsSeen = {};
        let recordedInput = 0, recordedOutput = 0, estimated = 0;
        for (const r of tok.rows) {
          byOrigin[r.token_origin] = (byOrigin[r.token_origin] || 0) + 1;
          byMethod[r.token_method] = (byMethod[r.token_method] || 0) + 1;
          if (r.reported_usage_fields) fieldsSeen[JSON.stringify(r.reported_usage_fields)] = 1;
          if (r.token_origin === "recorded_usage") { recordedInput += (r.input_tokens || 0); recordedOutput += (r.output_tokens || 0); }
          if (r.token_method === "round_utf8_bytes_div_4") estimated += (r.estimated_tokens || 0);
        }
        tokenEvidence = {
          row_count: tok.rows.length,
          token_origin_counts: byOrigin,
          token_method_counts: byMethod,
          reported_usage_fields_variants: Object.keys(fieldsSeen).map(s => JSON.parse(s)),
          recorded_input_tokens: recordedInput,
          recorded_output_tokens: recordedOutput,
          estimated_tokens_total: estimated,
          token_unit: (tok.rows.find(r => r.token_unit) || {}).token_unit ?? null,
        };
      }
      out.push({ framework: fw, archive: path.split("/").pop(), session_id: sess ? sess.session_id : null, agent_framework: sess ? sess.agent_framework : null, title: sess ? sess.title : null, turn_count: sess ? sess.turn_count : null, tool_count: toolCount, tool_call_count_meta: sess ? sess.tool_call_count : null, token_evidence: tokenEvidence ? JSON.stringify(tokenEvidence) : null, token_error: tok.error ? String(tok.error) : null });
    } finally { db.close(); }
  }
  return out;
}

__verb__("realtask", {
  name: "realtask",
  short: "Real archive summary per framework",
  fields: {},
});
