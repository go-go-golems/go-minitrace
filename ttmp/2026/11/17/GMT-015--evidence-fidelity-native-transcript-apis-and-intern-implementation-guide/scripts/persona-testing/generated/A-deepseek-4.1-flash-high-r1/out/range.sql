SELECT MIN(started_at) AS min_started, MAX(COALESCE(ended_at, started_at)) AS max_ended,
       COUNT(*) AS session_rows FROM sessions;
