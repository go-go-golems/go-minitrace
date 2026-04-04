-- 03-duckdb-sqlite-attach-attach-to-db.sql
-- Attach a DuckDB session to a new SQLite database via sqlite_attach
-- sqlite_attach(db_path) -> schema name defaults to filename (without .db extension)

INSTALL sqlite_scanner;
LOAD sqlite_scanner;

-- WARNING: This fails if the path doesn't exist. sqlite_attach creates the
-- schema but NOT the underlying database file. Must exist beforehand.
-- (This is why the first test failed with "unable to open database file")
-- sqlite_attach('/tmp/test-annotations.db');

-- Check the sqlite_attach documentation / examples
SELECT
    function_name,
    parameters,
    parameter_types,
    examples
FROM duckdb_functions()
WHERE function_name = 'sqlite_attach';
