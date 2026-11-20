SELECT agent_framework,
       COUNT(*) AS sessions,
       MIN(started_at) AS first_started,
       MAX(COALESCE(NULLIF(ended_at,''), started_at)) AS last_activity
FROM sessions
GROUP BY agent_framework
ORDER BY agent_framework;
