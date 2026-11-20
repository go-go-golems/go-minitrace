SELECT s.session_id, SUM(t.input_tokens) AS input_tokens, SUM(t.output_tokens) AS output_tokens,
       COUNT(t.input_tokens) AS input_turns, COUNT(t.output_tokens) AS output_turns,
       COUNT(t.turn_index) AS turns
FROM sessions s LEFT JOIN turns t ON s.session_id = t.session_id
GROUP BY s.session_id ORDER BY s.session_id;
