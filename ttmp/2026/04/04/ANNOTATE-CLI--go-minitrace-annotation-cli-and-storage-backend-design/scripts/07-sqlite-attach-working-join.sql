-- 07-sqlite-attach-working-join.sql
-- WORKING end-to-end test: attach SQLite annotations.db to DuckDB, join with sessions.
-- Run with: duckdb /tmp/sessions.duckdb < 07-sqlite-attach-working-join.sql
--
-- Key findings:
-- 1. sqlite_attach('/path/to/file.db', overwrite => true) — use NAMED parameter
-- 2. SQLite tables land in the main schema — query as just `annotations`, no schema prefix
-- 3. Schema name = filename without extension, but tables merge into main (not a sub-schema)
-- 4. No need for read_json on annotations — SQLite IS the annotations store
-- 5. sessions_base is loaded separately (via load.sql's read_json on .minitrace.json files)

INSTALL sqlite_scanner;
LOAD sqlite_scanner;

-- Attach the SQLite annotations DB to the DuckDB session.
-- overwrite=true allows re-attaching if the DB is already open.
CALL sqlite_attach('/tmp/test-annotations.db', overwrite => true);

-- SQLite tables are in main schema alongside DuckDB tables
SELECT
    id,
    session_id,
    category,
    title,
    scope_type,
    annotator,
    created_at,
    taxonomy_m
FROM annotations
ORDER BY created_at;

-- Create a sessions_base temp table (normally loaded via load.sql from .minitrace.json)
CREATE TEMP TABLE sessions_base AS
SELECT
    'sess-001' as id,
    '{"agent_framework":"claude-code","model":"claude-opus-4"}'::JSON as environment,
    'Spec review for external sharing' as title,
    'claude-code-jsonl-v2' as source_format,
    '{"tool_call_count": 47}'::JSON as metrics
UNION ALL
SELECT
    'sess-002' as id,
    '{"agent_framework":"codex","model":"gpt-4o"}'::JSON as environment,
    'Test session 2' as title,
    'codex-session-jsonl-v1' as source_format,
    '{"tool_call_count": 12}'::JSON as metrics;

-- Cross-session annotation query: join SQLite annotations with DuckDB sessions
SELECT
    a.session_id,
    sb.title as session_title,
    sb.source_format as framework,
    a.category,
    a.title as annotation_title,
    a.scope_type,
    a.annotator,
    a.created_at,
    a.taxonomy_m as taxonomy_minitrace,
    json_extract(sb.metrics, '$.tool_call_count') as tool_calls
FROM annotations a
JOIN sessions_base sb ON sb.id = a.session_id
ORDER BY a.session_id, a.created_at;

-- Annotation breakdown by session
SELECT
    a.session_id,
    sb.title as session_title,
    a.category,
    COUNT(*) as count
FROM annotations a
JOIN sessions_base sb ON sb.id = a.session_id
GROUP BY a.session_id, sb.title, a.category
ORDER BY a.session_id, count DESC;

-- Cross-session failure analysis
SELECT
    a.taxonomy_m as failure_codes,
    a.category,
    sb.environment->>'agent_framework' as framework,
    a.annotator
FROM annotations a
JOIN sessions_base sb ON sb.id = a.session_id
WHERE a.category = 'ai-failure'
ORDER BY a.created_at;
