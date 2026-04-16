-- file-timeline: Chronological operations on files matching a path pattern
-- Result content is classified into short labels for quick scanning
SELECT
  CAST(json_extract(tc, '$.emitting_turn_index') AS INT) AS turn,
  REPLACE(CAST(json_extract(tc, '$.tool_name') AS VARCHAR), '"', '') AS tool,
  json_extract(tc, '$.operation_type') AS operation,
  COALESCE(
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR),
    LEFT(CAST(json_extract(tc, '$.input.command') AS VARCHAR), 120)
  ) AS target,
  json_extract(tc, '$.output.success') AS success,
  CASE
    WHEN CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%File not found%' THEN 'FILE_NOT_FOUND'
    WHEN CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%Successfully%' THEN 'OK'
    WHEN CAST(json_extract(tc, '$.output.result') AS VARCHAR) LIKE '%No such file%' THEN 'NO_SUCH_FILE'
    WHEN CAST(json_extract(tc, '$.output.error') AS VARCHAR) != 'null'
      AND CAST(json_extract(tc, '$.output.error') AS VARCHAR) IS NOT NULL
    THEN 'ERROR'
    ELSE LEFT(CAST(json_extract(tc, '$.output.result') AS VARCHAR), 80)
  END AS result_summary,
  LEFT(CAST(json_extract(tc, '$.output.error') AS VARCHAR), 200) AS error,
  json_extract(tc, '$.timestamp') AS timestamp
FROM {{TABLE_NAME}},
  UNNEST(tool_calls) AS t(tc)
WHERE
  COALESCE(
    CAST(json_extract(tc, '$.input.file_path') AS VARCHAR),
    CAST(json_extract(tc, '$.input.command') AS VARCHAR), ''
  ) LIKE '%'
ORDER BY CAST(json_extract(tc, '$.emitting_turn_index') AS INT);
