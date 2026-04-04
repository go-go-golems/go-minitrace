# DuckDB SQLite Extension Investigation Scripts

Scripts from investigating DuckDB's `sqlite_scanner` extension for the annotation store design.

## Scripts

| # | File | Purpose |
|---|------|---------|
| 00 | `00-duckdb-sqlite-extension-discovery.sql` | Check if sqlite_scanner is available in the installed DuckDB |
| 01 | `01-duckdb-sqlite-extension-load.sql` | Install and load sqlite_scanner |
| 02 | `02-duckdb-sqlite-extension-function-sig.sql` | Discover sqlite_attach function signature |
| 03 | `03-duckdb-sqlite-attach-attach-to-db.sql` | Test attaching a DuckDB session to a SQLite DB |
| 04 | `04-duckdb-sqlite-inline-table-create.sql` | DuckDB-native tables as fallback (pre sqlite_attach) |
| 05 | `05-sqlite-schema.sql` | Raw SQLite schema for annotations.db |
| 05 | `05-sqlite-attach-and-query.sql` | Notes on sqlite_attach lifecycle |
| 06 | `06-sqlite-attach-and-join.sql` | Full test with real SQLite file (incorrect positional args) |
| 07 | `07-sqlite-attach-working-join.sql` | **WORKING** end-to-end test |

## Key Findings

1. `sqlite_scanner` is **built into DuckDB** — no external dependency needed
2. `INSTALL sqlite_scanner; LOAD sqlite_scanner;` required before use
3. `sqlite_attach` uses **named parameters**: `CALL sqlite_attach('/path', overwrite => true)`
4. Attached SQLite tables land in the `main` schema — query as `SELECT * FROM annotations`
5. **Annotations are live** — DuckDB queries SQLite directly, no refresh needed after writes
6. The `annotations_flat` JSON export approach (earlier design) is **not needed**

## Running

```bash
# Test the working end-to-end flow
duckdb /tmp/sessions.duckdb < scripts/07-sqlite-attach-working-join.sql

# Create the SQLite DB
sqlite3 /tmp/test-annotations.db < scripts/05-sqlite-schema.sql

# Insert test data
sqlite3 /tmp/test-annotations.db "INSERT INTO annotations VALUES (...);"
```
