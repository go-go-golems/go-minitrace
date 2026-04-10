-- Tool frequency: balance of reading vs writing vs building
SELECT
  json_extract(tc, '$.tool_name') AS tool_name,
  COUNT(*) AS calls
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
GROUP BY tool_name
ORDER BY calls DESC;
