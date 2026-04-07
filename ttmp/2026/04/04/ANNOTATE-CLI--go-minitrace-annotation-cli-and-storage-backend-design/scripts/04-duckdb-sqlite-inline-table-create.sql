-- 04-duckdb-sqlite-inline-table-create.sql
-- Test: create SQLite-like tables inline in DuckDB and query them
-- This shows the schema and data layout that the annotation store uses

INSTALL sqlite_scanner;
LOAD sqlite_scanner;

-- The sqlite_scanner attaches a SQLite DB as a schema.
-- If the DB doesn't exist, it CANNOT create it (DB must exist on disk first).
-- Alternative: use DuckDB native tables that mirror the SQLite schema.
-- This is what the annotations_flat approach does.

-- Create a DuckDB-native table that matches the SQLite schema
CREATE TABLE annotations_flat (
    id VARCHAR PRIMARY KEY,
    session_id VARCHAR NOT NULL,
    annotator VARCHAR NOT NULL,
    scope_type VARCHAR NOT NULL,
    target_id VARCHAR NOT NULL,
    category VARCHAR NOT NULL,
    title VARCHAR NOT NULL,
    detail VARCHAR NOT NULL DEFAULT '',
    tags VARCHAR NOT NULL DEFAULT '[]',
    taxonomy_minitrace VARCHAR NOT NULL DEFAULT '[]',
    taxonomy_mast VARCHAR NOT NULL DEFAULT '[]',
    taxonomy_toolemu VARCHAR NOT NULL DEFAULT '[]',
    classification VARCHAR,
    created_at VARCHAR NOT NULL,
    updated_at VARCHAR NOT NULL
);

-- Insert test data
INSERT INTO annotations_flat VALUES
    ('ann-1', 'sess-001', 'user', 'session', 'sess-001', 'ai-failure', 'Over-autonomy',
     'Acted beyond scope', '["F-AUT"]', '["F-AUT"]', '[]', '[]', NULL,
     '2026-04-04T12:00:00Z', '2026-04-04T12:00:00Z'),
    ('ann-2', 'sess-001', 'user', 'turn', 'sess-001', 'observation', 'Late session behavior',
     'C-PHA context code noted', '["C-PHA"]', '["C-PHA"]', '[]', '[]', NULL,
     '2026-04-04T12:05:00Z', '2026-04-04T12:05:00Z'),
    ('ann-3', 'sess-002', 'automated', 'session', 'sess-002', 'pattern', 'Low read ratio',
     'Read ratio < 0.2 across framework', '[]', '[]', '[]', '[]', NULL,
     '2026-04-04T13:00:00Z', '2026-04-04T13:00:00Z');

-- Query all
SELECT id, session_id, category, title FROM annotations_flat ORDER BY id;

-- Clean up
DROP TABLE annotations_flat;
