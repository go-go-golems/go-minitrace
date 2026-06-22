__package__({
  name: "inspect",
  short: "Inspect one .minitrace.json archive with require(\"minitrace\")",
});

function openDB(file) {
  const mt = require("minitrace");
  return mt.db().File(file).QueryCommandDefaults().Build();
}

function summary(file) {
  const db = openDB(file);
  try {
    const row = db.queryOne(`
      SELECT
        session_id,
        title,
        agent_framework,
        working_directory,
        turn_count,
        tool_call_count,
        started_at
      FROM sessions
      ORDER BY session_id
      LIMIT 1
    `);
    return {
      file,
      session_id: row.session_id,
      title: row.title,
      framework: row.agent_framework,
      working_directory: row.working_directory,
      turns: row.turn_count,
      tools: row.tool_call_count,
      started_at: row.started_at,
    };
  } finally {
    db.close();
  }
}

__verb__("summary", {
  name: "summary",
  short: "Print high-level metadata for one minitrace archive",
  fields: {
    file: {
      argument: true,
      help: "Path to a .minitrace.json archive",
    },
  },
});

function tools(file, limit) {
  const db = openDB(file);
  try {
    return db.query(`
      SELECT
        tool_name,
        COALESCE(operation_type, '') AS operation_type,
        COUNT(*) AS calls,
        SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) AS failures
      FROM tool_calls
      GROUP BY tool_name, operation_type
      ORDER BY calls DESC, tool_name ASC, operation_type ASC
      LIMIT ?
    `, limit || 10);
  } finally {
    db.close();
  }
}

__verb__("tools", {
  name: "tools",
  short: "Summarize tool-call counts for one minitrace archive",
  fields: {
    file: {
      argument: true,
      help: "Path to a .minitrace.json archive",
    },
    limit: {
      type: "int",
      default: 10,
      help: "Maximum number of tool rows to return",
    },
  },
});
