-- Rewrite timestamps: when churn happened
SELECT
  json_extract(tc, '$.input.file_path') AS file_path,
  json_extract(tc, '$.tool_name') AS tool,
  COUNT(*) AS times_touched,
  MIN(json_extract(tc, '$.timestamp')) AS first_touch,
  MAX(json_extract(tc, '$.timestamp')) AS last_touch
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE json_extract(tc, '$.tool_name') IN ('"write"', '"edit"')
  AND json_extract(tc, '$.input.file_path') IS NOT NULL
GROUP BY file_path, tool
HAVING COUNT(*) > 1
ORDER BY times_touched DESC;
