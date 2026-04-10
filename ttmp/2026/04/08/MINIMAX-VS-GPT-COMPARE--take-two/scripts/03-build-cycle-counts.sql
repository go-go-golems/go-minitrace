-- Build/test cycle count
SELECT
  CASE
    WHEN CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%go build%' THEN 'go-build'
    WHEN CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%go test%' THEN 'go-test'
    WHEN CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%go run%' THEN 'go-run'
    WHEN CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%pnpm%' THEN 'pnpm'
    ELSE 'other'
  END AS cmd_type,
  COUNT(*) AS count
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE json_extract(tc, '$.tool_name') = '"bash"'
  AND (
    CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%go build%'
    OR CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%go test%'
    OR CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%go run%'
    OR CAST(json_extract(tc, '$.input.command') AS VARCHAR) LIKE '%pnpm%'
  )
GROUP BY cmd_type
ORDER BY count DESC;
