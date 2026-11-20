// overview session-activity — list sessions ordered by when they were last
// touched, where "touched" means the latest of any turn or any tool call.
//
// Split out of the skill's combined session-activity.js (which also defined
// file-activity) so that this file declares a single verb. The embedded
// catalog collapses a single-verb file's path, so this is reachable as
// `query commands overview session-activity` rather than the doubled
// `overview/session-activity/session-activity` a two-verb file would produce.

__section__("filters", {
  title: "Session activity filters",
  fields: {
    framework: {
      type: "stringList",
      help: "Frameworks to include, for example pi,codex,claude-code",
    },
    cwd_contains: {
      type: "string",
      help: "Case-sensitive working-directory substring",
    },
    since: {
      type: "string",
      help: "Only activity at or after this RFC3339 timestamp",
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
    const result = db.execute(mt.sql`
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
        AND (${frameworks.length === 0} OR (s.framework IS NOT NULL AND instr(${keys}, '|' || hex(s.framework) || '|') > 0))
        AND (${cwd} IS NULL OR s.working_directory LIKE ${cwd})
        AND (${since} IS NULL OR last_activity_at >= ${since})
      ORDER BY last_activity_at DESC
      LIMIT ${filters.limit ?? 100}
    `.build());
    if (result.error || !result.complete) throw new Error(result.error || "Incomplete query result");
    return result.rows;
  } finally {
    db.close();
  }
}

__verb__("sessionActivity", {
  name: "session-activity",
  short: "List sessions by last interaction time",
  fields: { filters: { bind: "filters" } },
});
