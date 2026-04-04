-- 00-duckdb-sqlite-extension-discovery.sql
-- Check if DuckDB has sqlite_scanner extension built-in

SELECT extension_name, loaded
FROM duckdb_extensions()
WHERE extension_name IN ('sqlite', 'sqlite_scanner');
