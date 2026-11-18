const mt = require("minitrace");

const minTools = mt.fragment({
  name: "minimum-tools",
  target: "query",
  requires: ["tool_count"],
  methods: {
    minTools(q, min) {
      return q.where(r => r.tool_count.gte(min));
    },
  },
});

function report() {
  const sources = mt.sources().runtimeArchives().build();
  const db = mt.db().sources(sources)
    .import({ autoConvert: true, strict: true })
    .limits({ maxRows: 2000, maxCellBytes: 8192, strict: true })
    .build();
  try {
    const sessions = mt.from(db.table("sessions"));
    const tools = mt.from(db.table("tool_calls"));

    const counts = sessions
      .extend(minTools)
      .select(s => ({
        session_id: s.session_id,
        tool_count: tools.where(t => t.session_id.eq(s.session_id)).count(),
      }))
      .minTools(0)
      .orderBy(s => s.session_id.asc());

    const result = db.execute(counts.build());
    if (result.error || !result.complete) {
      throw Error(result.error || "Incomplete result");
    }
    return result.rows;
  } finally {
    db.close();
  }
}

__verb__("report", { name: "report" });
