// files file-activity — list touched files by their most recent write or tool
// activity, one row per (session, file) with an operation count.
//
// Split out of the skill's combined overview/session-activity.js so that this
// file declares a single verb (the embedded catalog collapses a single-verb
// file's path). Filed under files/ rather than overview/ to sit alongside the
// other file-oriented commands, file-operations.sql and file-timeline.sql.

__section__("filters", {
  title: "File activity filters",
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
      help: "Include only NEW and MODIFY operations",
    },
    limit: {
      type: "int",
      default: 100,
      help: "Maximum rows to return",
    },
  },
});

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
      ? "AND (CASE WHEN s.agent_framework='codex' THEN f.operation_type ELSE tc.operation_type END) IN ('NEW', 'MODIFY', 'DELETE')"
      : "";

    // `operations` counts rows surviving the ranked WHERE, so narrowing with
    // --since or --path-contains narrows the count too: it reports operations
    // within the requested window, not the file's lifetime total.
    return db.query(`
      WITH calls AS (
        SELECT
          tc.session_id,
          tc.timestamp,
          tc.tool_name,
          CASE WHEN s.agent_framework='codex' THEN f.operation_type ELSE tc.operation_type END AS operation_type,
          f.evidence_status,
          CASE WHEN s.agent_framework='codex' THEN f.path ELSE COALESCE(
            NULLIF(tc.file_path, ''),
            json_extract(tc.arguments_json, '$.path'),
            json_extract(tc.arguments_json, '$.file_path')
          ) END AS file_path
        FROM tool_calls tc
        JOIN sessions s USING (session_id)
        LEFT JOIN files f ON s.agent_framework='codex' AND f.session_id=tc.session_id AND f.tool_call_id=tc.tool_call_id AND f.evidence_kind!='legacy_scalar'
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
          evidence_status AS latest_evidence_status,
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
             latest_operation, latest_tool, latest_evidence_status
      FROM ranked
      WHERE row_number = 1
      ORDER BY last_activity_at DESC
      LIMIT ${filters.limit ?? 100}
    `);
  } finally {
    db.close();
  }
}

__verb__("fileActivity", {
  name: "file-activity",
  short: "List touched files by last write or tool activity",
  fields: { filters: { bind: "filters" } },
});
