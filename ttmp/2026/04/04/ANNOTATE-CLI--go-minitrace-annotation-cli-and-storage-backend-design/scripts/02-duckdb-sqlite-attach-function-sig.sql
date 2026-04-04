-- 02-duckdb-sqlite-attach-function-sig.sql
-- Check sqlite_attach function signature

INSTALL sqlite_scanner;
LOAD sqlite_scanner;

SELECT
    function_name,
    parameters,
    parameter_types,
    schema_name
FROM duckdb_functions()
WHERE function_name = 'sqlite_attach';
