-- Actual output of the chosen candidate (pi session 01a0b061..., read of playback/CLAUDE.md)
SELECT substr(t.result,1,700) AS head, substr(t.result,-400) AS tail, t.arguments_json
FROM tool_calls t
WHERE t.session_id='01a0b061-0cdd-7557-a80a-96802d3d02c5'
  AND t.tool_call_id LIKE 'call_Kyn9Az5XTsNJLXQi7yVnBuMz%';
