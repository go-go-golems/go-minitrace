function counts() {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().SQLiteMemory().StrictEvidence(true).Build();
  try {
    const result = db.execute(mt.query().SessionToolCounts().Build());
    if (result.error || !result.complete) throw new Error(result.error || "Incomplete evidence");
    return result.rows;
  } finally { db.close(); }
}
__verb__("counts", {name: "counts", short: "Count actual tools including zero-tool sessions"});
