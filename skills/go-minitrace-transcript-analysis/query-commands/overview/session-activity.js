__section__("filters", {
  title: "Activity filters",
  fields: {
    framework: {
      type: "stringList",
      help: "Frameworks to include, for example pi,codex,claude-code",
    },
    cwd_contains: {
      type: "string",
      help: "Case-sensitive working-directory substring",
    },
    path_contains: {
      type: "string",
      help: "Case-sensitive file-path substring",
    },
    since: {
      type: "string",
      help: "Only activity at or after this RFC3339 timestamp",
    },
    write_only: {
      type: "bool",
      default: true,
      help: "For file activity, include only NEW and MODIFY operations",
    },
    limit: {
      type: "int",
      default: 100,
      help: "Maximum rows to return",
    },
  },
});

function sessionActivity(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    const framework = filters.framework?.length
      ? `AND s.agent_framework IN (${mt.sql.stringIn(filters.framework)})`
      : "";
    const cwd = filters.cwd_contains
      ? `AND s.working_directory LIKE ${mt.sql.like(`%${filters.cwd_contains}%`)}`
      : "";
    const since = filters.since
      ? `AND last_activity_at >= ${mt.sql.string(filters.since)}`
      : "";

    // Last activity is the max timestamp over turns AND tool_calls, computed
    // by unioning the two timestamp streams and aggregating once per session.
    //
    // The previous form joined both tables to `sessions` and took
    // MAX(COALESCE(t.timestamp, tc.timestamp)). That was wrong twice over.
    // COALESCE is per-row and returns its first non-null argument, so once a
    // session had any turns at all every joined row carried a turn timestamp
    // and the tool_call timestamps were never considered — a session whose
    // last tool call ran after its last turn reported the stale turn time.
    // Joining both tables also built a turns x tool_calls cross product per
    // session: 500 turns and 800 tool calls meant 400,000 rows scanned to
    // compute a single maximum.
    return db.query(`
      WITH stamps AS (
        SELECT session_id, timestamp FROM turns
        UNION ALL
        SELECT session_id, timestamp FROM tool_calls
      )
      , last_seen AS (
        SELECT session_id, MAX(timestamp) AS last_activity_at
        FROM stamps
        WHERE timestamp IS NOT NULL AND timestamp <> ''
        GROUP BY session_id
      )
      , activity AS (
        SELECT
          s.session_id,
          s.agent_framework AS framework,
          s.model,
          s.title,
          s.working_directory,
          s.started_at,
          s.ended_at,
          s.turn_count,
          s.tool_call_count,
          l.last_activity_at
        FROM sessions s
        LEFT JOIN last_seen l USING (session_id)
      )
      SELECT *
      FROM activity s
      WHERE 1=1
        ${framework}
        ${cwd}
        ${since}
      ORDER BY last_activity_at DESC
      LIMIT ${filters.limit ?? 100}
    `);
  } finally {
    db.close();
  }
}

function fileActivity(filters) {
  const mt = require("minitrace");
  const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build();
  try {
    const framework = filters.framework?.length
      ? `AND s.agent_framework IN (${mt.sql.stringIn(filters.framework)})`
      : "";
    const cwd = filters.cwd_contains
      ? `AND s.working_directory LIKE ${mt.sql.like(`%${filters.cwd_contains}%`)}`
      : "";
    const path = filters.path_contains
      ? `AND file_path LIKE ${mt.sql.like(`%${filters.path_contains}%`)}`
      : "";
    const since = filters.since
      ? `AND timestamp >= ${mt.sql.string(filters.since)}`
      : "";
    const writes = filters.write_only !== false
      ? "AND tc.operation_type IN ('NEW', 'MODIFY')"
      : "";

    return db.query(`
      WITH calls AS (
        SELECT
          tc.session_id,
          tc.timestamp,
          tc.tool_name,
          tc.operation_type,
          COALESCE(
            NULLIF(tc.file_path, ''),
            json_extract(tc.arguments_json, '$.path'),
            json_extract(tc.arguments_json, '$.file_path')
          ) AS file_path
        FROM tool_calls tc
        JOIN sessions s USING (session_id)
        WHERE 1=1
          ${framework}
          ${cwd}
          ${writes}
      )
      , ranked AS (
        SELECT
          session_id,
          file_path,
          timestamp AS last_activity_at,
          operation_type AS latest_operation,
          tool_name AS latest_tool,
          COUNT(*) OVER (PARTITION BY session_id, file_path) AS operations,
          ROW_NUMBER() OVER (
            PARTITION BY session_id, file_path
            ORDER BY timestamp DESC
          ) AS row_number
        FROM calls
        WHERE file_path IS NOT NULL
          AND file_path <> ''
          ${path}
          ${since}
      )
      SELECT session_id, file_path, last_activity_at, operations,
             latest_operation, latest_tool
      FROM ranked
      WHERE row_number = 1
      ORDER BY last_activity_at DESC
      LIMIT ${filters.limit ?? 100}
    `);
  } finally {
    db.close();
  }
}

__verb__("sessionActivity", {
  name: "session-activity",
  short: "List sessions by last interaction time",
  fields: { filters: { bind: "filters" } },
});

__verb__("fileActivity", {
  name: "file-activity",
  short: "List touched files by last write or tool activity",
  fields: { filters: { bind: "filters" } },
});
