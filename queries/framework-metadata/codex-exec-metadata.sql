-- codex-exec-metadata: Inspect preserved Codex execution metadata
-- Shows command source, parsed command info, stdout/stderr, and exit codes
-- Usage:
--   duckdb analysis.duckdb -init queries/load.sql -f queries/framework-metadata/codex-exec-metadata.sql

SELECT
  sb.id AS session_id,
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  CAST(json_extract(tc, '$.input.command') AS VARCHAR) AS command,
  CAST(json_extract(tc, '$.output.exit_code') AS INT) AS exit_code,
  CAST(json_extract(tc, '$.framework_metadata.source') AS VARCHAR) AS source,
  json_extract(tc, '$.framework_metadata.parsed_cmd') AS parsed_cmd,
  CAST(json_extract(tc, '$.framework_metadata.stdout') AS VARCHAR) AS stdout,
  CAST(json_extract(tc, '$.framework_metadata.stderr') AS VARCHAR) AS stderr,
  json_extract(tc, '$.timestamp') AS timestamp
FROM sessions_base AS sb,
  UNNEST(tool_calls) AS t(tc)
WHERE
  CAST(json_extract(sb.environment, '$.agent_framework') AS VARCHAR) = '"codex"'
  AND json_extract(tc, '$.framework_metadata.source') IS NOT NULL
ORDER BY
  sb.id,
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT),
  json_extract(tc, '$.timestamp');
