-- 06-sqlite-attach-and-join.sql
-- End-to-end test: create SQLite DB, attach to DuckDB, join with session data.
-- Run with: duckdb /path/to/sessions.duckdb < 06-sqlite-attach-and-join.sql
-- Assumes: annotations.db exists at /tmp/annotations.db with the schema from 05-sqlite-schema.sql

INSTALL sqlite_scanner;
LOAD sqlite_scanner;

-- First, create a minimal SQLite DB for testing
-- (In practice: the Go code creates this via go-sqlite3)

-- Attach the SQLite annotations DB to the DuckDB session.
-- The schema name is derived from the filename: "annotations" from "annotations.db"
-- If overwrite=true, it re-attaches (useful for reloading after changes).
CALL sqlite_attach('/tmp/annotations.db', true);

-- Verify: can we query the attached SQLite tables?
-- If annotations.db has the schema from 05-sqlite-schema.sql:
-- SELECT COUNT(*) FROM annotations;

-- Simulate the annotations_flat approach: query SQLite, then join with DuckDB sessions
-- Step 1: Load annotations from SQLite (as DuckDB-native data)
CREATE TEMP TABLE annotations_from_sqlite AS
SELECT
    id,
    session_id,
    annotator,
    scope_type,
    target_id,
    category,
    title,
    detail,
    tags,
    taxonomy_minitrace,
    taxonomy_mast,
    taxonomy_toolemu,
    classification,
    created_at,
    updated_at
FROM annotations  -- this is the SQLite-attached table
ORDER BY created_at;

-- Step 2: Join with session data (assumes sessions_base is loaded)
-- SELECT
--     af.session_id,
--     sb.environment->>'agent_framework' AS framework,
--     sb.title AS session_title,
--     af.category,
--     af.title AS annotation_title,
--     af.annotator,
--     af.taxonomy_minitrace
-- FROM annotations_from_sqlite af
-- JOIN sessions_base sb ON sb.id = af.session_id
-- ORDER BY af.created_at;

-- The key insight: this approach reads from SQLite every time the query runs.
-- No need to refresh a temp table. SQLite IS the source of truth.
-- DuckDB just queries it like any other table.

-- Cleanup
DROP TABLE IF EXISTS annotations_from_sqlite;
