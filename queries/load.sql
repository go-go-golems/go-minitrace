-- load.sql
-- Load converted minitrace sessions into a temp table once per DuckDB session.
-- Usage:
--   duckdb analysis.duckdb -init queries/load.sql
-- or inside duckdb:
--   .read queries/load.sql

CREATE OR REPLACE TEMP TABLE sessions_base AS
SELECT *
FROM read_json(
  './output/active/*/*.minitrace.json',
  columns = {
    id: 'VARCHAR',
    title: 'VARCHAR',
    summary: 'VARCHAR',
    classification: 'VARCHAR',
    profile: 'VARCHAR',
    provenance: 'JSON',
    flags: 'JSON',
    environment: 'JSON',
    operational_context: 'JSON',
    timing: 'JSON',
    turns: 'JSON[]',
    tool_calls: 'JSON[]',
    annotations: 'JSON[]',
    metrics: 'JSON'
  },
  ignore_errors = true
);
