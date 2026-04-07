-- 01-duckdb-sqlite-extension-load.sql
-- Install and load the sqlite_scanner extension

INSTALL sqlite_scanner;
LOAD sqlite_scanner;

SELECT extension_name, loaded
FROM duckdb_extensions()
WHERE extension_name = 'sqlite_scanner';
