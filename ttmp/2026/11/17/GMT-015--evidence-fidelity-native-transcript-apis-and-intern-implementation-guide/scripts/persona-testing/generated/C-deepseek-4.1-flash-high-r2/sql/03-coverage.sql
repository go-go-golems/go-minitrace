-- Normalized coverage per framework (why zero native api-request != zero instructions).
SELECT agent_framework AS fw, COUNT(*) AS sessions,
       SUM(turn_count) AS turns, SUM(tool_call_count) AS tools
FROM sessions GROUP BY 1;
