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
  const db = mt.db().sources(mt.sources().runtimeArchives().build())
    .import({autoConvert: true, strict: true}).build();
  try {
    const frameworks = filters.framework || [];
    // Frame UTF-8 hex keys for exact membership with a single bound value.
    const keys = "|" + frameworks.map(value => encodeURIComponent(value)
      .replace(/%[0-9A-F]{2}|./g, token => token[0] === "%"
        ? token.slice(1)
        : token.charCodeAt(0).toString(16).padStart(2, "0").toUpperCase())).join("|") + "|";
    const cwd = filters.cwd_contains ? "%" + filters.cwd_contains + "%" : null;
    const since = filters.since || null;
    const path = filters.path_contains ? "%" + filters.path_contains + "%" : null;
    const writesOnly = filters.write_only !== false;

    // `operations` counts rows surviving the ranked WHERE, so narrowing with
    // --since or --path-contains narrows the count too: it reports operations
    // within the requested window, not the file's lifetime total.
    const result = db.execute(mt.sql`
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
          AND (${frameworks.length === 0} OR (s.agent_framework IS NOT NULL AND instr(${keys}, '|' || hex(s.agent_framework) || '|') > 0))
          AND (${cwd} IS NULL OR s.working_directory LIKE ${cwd})
          AND (NOT ${writesOnly} OR (CASE WHEN s.agent_framework='codex' THEN f.operation_type ELSE tc.operation_type END) IN ('NEW', 'MODIFY', 'DELETE'))
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
          AND (${path} IS NULL OR file_path LIKE ${path})
          AND (${since} IS NULL OR timestamp >= ${since})
      )
      SELECT session_id, file_path, last_activity_at, operations,
             latest_operation, latest_tool, latest_evidence_status
      FROM ranked
      WHERE row_number = 1
      ORDER BY last_activity_at DESC
      LIMIT ${filters.limit ?? 100}
    `.build());
    if (result.error || !result.complete) throw new Error(result.error || "Incomplete query result");
    return result.rows;
  } finally {
    db.close();
  }
}

__verb__("fileActivity", {
  name: "file-activity",
  short: "List touched files by last write or tool activity",
  fields: { filters: { bind: "filters" } },
});
