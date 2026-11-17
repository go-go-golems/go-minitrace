-- -- (A) READ evidence: pi tool "read" with file_path + captured output.
SELECT session_id, emitting_turn_index AS turn, tool_name, operation_type, file_path,
       success, truncated, full_bytes, full_hash, length(result) AS result_len,
       substr(result,1,1200) AS result_head
FROM tool_calls
WHERE session_id LIKE '01a0b061%' AND emitting_turn_index=3
  AND file_path LIKE '%playback/CLAUDE.md';

