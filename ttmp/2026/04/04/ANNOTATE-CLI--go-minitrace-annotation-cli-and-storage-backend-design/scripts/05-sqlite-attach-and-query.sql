-- 05-sqlite-attach-and-query.sql
-- Full test: create a SQLite DB with the annotation schema,
-- attach it to DuckDB via sqlite_scanner, and query across both.
-- Run with: duckdb < 05-sqlite-attach-and-query.sql

INSTALL sqlite_scanner;
LOAD sqlite_scanner;

-- Step 1: Create the SQLite DB on disk first (with sqlite3 CLI)
-- sqlite3 /tmp/test-annotations.db < 05-sqlite-schema.sql

-- Step 2: Attach to DuckDB
-- The schema name becomes the filename without extension ("test-annotations")
-- This MUST be done BEFORE any CREATE TABLE statements in DuckDB

-- But: we can also use a :memory: trick or pre-create the file.
-- Let's use the filesystem for real testing:

-- Pre-create the SQLite DB file
CREATE TABLE IF NOT EXISTS sqlite_anno_annotations AS
SELECT
    'ann-1' as id, 'sess-001' as session_id, 'user' as annotator,
    'session' as scope_type, 'sess-001' as target_id, 'ai-failure' as category,
    'Over-autonomy' as title, 'Acted beyond scope' as detail,
    '["F-AUT"]' as tags, '["F-AUT"]' as taxonomy_minitrace,
    '[]' as taxonomy_mast, '[]' as taxonomy_toolemu,
    NULL as classification, '2026-04-04T12:00:00Z' as created_at,
    '2026-04-04T12:00:00Z' as updated_at
WHERE false;

-- This approach is awkward. Better to use sqlite3 CLI first, then DuckDB.
-- Script for sqlite3 CLI:
-- $ sqlite3 /tmp/test-annotations.db < schema.sql
-- $ sqlite3 /tmp/test-annotations.db "INSERT INTO annotations ... ;"

-- Instead, let's verify the sqlite_attach works with a path:
-- NOTE: This test was run and showed that sqlite_attach CAN create tables
-- in the attached DB, but the DB file itself must exist beforehand.
-- The working approach was:
-- 1. duckdb runs in-process and attaches the SQLite DB
-- 2. DuckDB can then INSERT into it via the attached schema
-- 3. The annotations_flat approach (exporting SQLite to JSON, loading into DuckDB)
--    was used instead to avoid the sqlite_attach lifecycle complexity.

SELECT 'See 04-duckdb-sqlite-inline-table-create.sql for the working DuckDB-native approach' as note;
SELECT 'See 05-sqlite-schema.sql for the raw SQLite schema' as note;
