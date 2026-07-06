---
Title: Writing DuckDB Queries (removed engine)
Slug: writing-duckdb-queries
Short: The DuckDB engine was removed; see writing-queries for the normalized SQLite tutorial
Topics:
- minitrace
- sqlite
IsTemplate: false
IsTopLevel: false
ShowPerDefault: false
SectionType: Tutorial
---

The DuckDB backend this tutorial used to target has been removed from
go-minitrace. All SQL now runs on the normalized SQLite engine.

- For the current query-writing tutorial (normalized tables, `json_extract`
  patterns, filtering and aggregation recipes), see
  `go-minitrace help writing-queries`.
- For rewriting saved DuckDB SQL (UNNEST, `LEFT(...)`, `CAST(... AS DATE)`,
  arrow operators on blob columns), see the migration table in
  `go-minitrace help query-duckdb`.

Session-level `->>` examples written against the old `sessions_base` table
still work unmodified: the normalized database exposes a `sessions_base`
compatibility view with the same shape.
