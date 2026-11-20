-- Concrete actual-read candidate and its captured output size/head
SELECT session_id, emitting_turn_index AS turn, tool_name, file_path, success,
       length(json_extract(raw_json,'$.output.result')) AS output_bytes,
       substr(json_extract(raw_json,'$.output.result'),1,300) AS output_head,
       json_extract(raw_json,'$.input.command') AS input_command
FROM tool_calls
WHERE session_id='01a0ee23-bd68-75c1-a81f-2c43771fc216'
  AND tool_name='read' AND file_path='CLAUDE.md';
