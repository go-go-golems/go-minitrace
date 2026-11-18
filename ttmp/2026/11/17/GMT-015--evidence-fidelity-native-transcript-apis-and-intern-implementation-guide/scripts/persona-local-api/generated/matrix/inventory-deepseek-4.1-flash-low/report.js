const mt = require("minitrace");

function report() {
  const sources = mt.sources().runtimeArchives().build();
  const db = mt.db().sources(sources)
    .import({autoConvert: true, strict: true})
    .build();

  try {
    const sessions = mt.from(db.table("sessions"));
    const tools = mt.from(db.table("tool_calls"));

    const minimumToolFilter = mt.fragment({
      name: "minimum-tool-filter",
      target: "query",
      requires: ["tool_count"],
      methods: {
        atLeastTools(q, minimum) {
          return q.where(row => row.tool_count.gte(minimum));
        }
      }
    });

    const plan = sessions.select(s => ({
      session_id: s.session_id,
      tool_count: tools.where(t => t.session_id.eq(s.session_id)).count()
    }))
      .extend(minimumToolFilter)
      .atLeastTools(0)
      .orderBy(s => s.session_id.asc())
      .build();

    const result = db.execute(plan);
    if (result.error || !result.complete) {
      throw Error(result.error || "Incomplete result");
    }
    return result.rows;
  } finally {
    db.close();
  }
}

__verb__("report", {name: "report"});
