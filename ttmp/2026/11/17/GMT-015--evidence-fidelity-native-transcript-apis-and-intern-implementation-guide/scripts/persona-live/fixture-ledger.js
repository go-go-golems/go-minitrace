// Known-answer implementation for the offline harness check.
function ledger() {
  const mt = require("minitrace");
  const db = mt.db().sources(mt.sources().runtimeArchives().build())
    .limits({ maxRows: 10000, strict: true }).build();
  try {
    const result = db.execute(mt.sql`
      SELECT f.session_id, f.tool_call_id, f.target_ordinal, f.path, f.evidence_status
      FROM files f JOIN sessions s ON f.session_id = s.session_id
      WHERE f.operation_type IN ('NEW', 'MODIFY', 'DELETE')
        AND (COALESCE(s.agent_framework,'') != 'codex' OR f.evidence_kind != 'legacy_scalar')
      ORDER BY f.session_id, f.tool_call_id, f.target_ordinal, f.path
    `.build());
    if (!result.complete || result.error) throw Error(result.error || "Incomplete ledger");
    return result.rows;
  } finally {
    db.close();
  }
}
__verb__("ledger", { name: "ledger" });
