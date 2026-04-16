-- Dump a few write tool calls to see the actual input structure
SELECT
  tc.emitting_turn_index AS turn,
  json_structure(tc.input) AS input_structure,
  json_structure(tc.output) AS output_structure
FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE CAST(tc.tool_name AS VARCHAR) = 'write'
LIMIT 2;
