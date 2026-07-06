---
Title: DuckDB Query Recipes (removed engine)
Slug: duckdb-query-recipes
Short: The DuckDB engine was removed; see query-recipes for the normalized SQLite recipes
Topics:
- minitrace
- sqlite
IsTemplate: false
IsTopLevel: false
ShowPerDefault: false
SectionType: Example
---

The DuckDB backend these recipes used to target has been removed from
go-minitrace. All SQL now runs on the normalized SQLite engine.

- The recipes were ported to the normalized schema: see
  `go-minitrace help query-recipes`. Runnable copies also live in the repo
  `queries/` tree and the embedded presets (`go-minitrace query run --preset`).
- For rewriting your own saved DuckDB SQL, see the migration table in
  `go-minitrace help query-duckdb`.

Session-level recipes that used `->>` on the old `sessions_base` blob columns
still work unmodified against the `sessions_base` compatibility view; recipes
that used `UNNEST` must switch to the child tables (`tool_calls`, `turns`,
`annotations`, ...).
