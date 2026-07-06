---
Title: Migrating from query duckdb
Slug: query-duckdb
Short: The legacy DuckDB backend was removed; how to move saved SQL to query run
Topics:
- minitrace
- sqlite
- glazed
Commands:
- query run
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The legacy DuckDB backend (`go-minitrace query duckdb`) has been removed. All
SQL now runs on the normalized SQLite engine through a single sandboxed query
runner shared by `query run`, `.sql` query-command files, the JS runtime
(`mt.db()`), and `serve`.

## The replacement

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql 'SELECT session_id, title, turn_count FROM sessions ORDER BY started_at'
```

`query run` supports `--sql`, `--sql-file`, and `--preset`. The nine presets
were ported to the normalized schema (`session-list`, `framework-summary`,
`annotations`, `timing-analysis`, `tool-operation-breakdown`, `tool-failures`,
`read-ratio-distribution`, `file-operations`, `file-timeline`; see
`go-minitrace help query-commands` for descriptions). Limits are configurable
via `--max-rows`, `--max-cell-chars`, and `--timeout-ms`.

## Rewriting saved DuckDB SQL

The normalized schema pre-flattens what `sessions_base` kept as JSON blobs:

| DuckDB pattern | Normalized SQLite equivalent |
|---|---|
| `environment->>'agent_framework'` | `sessions.agent_framework` column |
| `metrics->>'turn_count'` | `sessions.turn_count` (more rollups in the `metrics` table) |
| `->` / `->>` on other blob columns | prefer the real `sessions` columns; unchanged SQL keeps working against the `sessions_base` compat view |
| `UNNEST(tool_calls) AS t(tc)` | query the `tool_calls` table, join `USING (session_id)` |
| `UNNEST(turns) WITH ORDINALITY` | query the `turns` table (`turn_index` column) |
| `LEFT(x, n)` | `substr(x, 1, n)` |
| `CAST(x AS DATE)` | `date(x)` |
| `REPLACE(CAST(json_extract(...) AS VARCHAR), '"', '')` | plain `json_extract(...)` — SQLite returns unquoted values |
| `DESCRIBE` / `SHOW TABLES` | `db.schema()` / `db.tables()` from JS (the sandbox blocks `sqlite_master`), or `go-minitrace help minitrace-schema` |
| `read_json(...)` | not needed — the database is built from archives automatically |

Session-level SQL using `->`/`->>` on the old blob columns keeps working
unmodified against the `sessions_base` compatibility view (SQLite >= 3.38
supports the JSON arrow operators); per-tool-call and per-turn SQL must move to
the `tool_calls`/`turns` tables. Long-tail fields that never got a real column
are reachable via `json_extract(raw_json, ...)` on every table.

Annotations are attached live by `serve` as `anno.annotations`; from the CLI,
the archive-synced annotations are in the `annotations` table (run
`go-minitrace annotate sync` first).

## See also

- `go-minitrace help writing-queries` — the current query-writing tutorial
- `go-minitrace help query-recipes` — ported, ready-to-run recipes
- `go-minitrace help query-commands` — `query run` flags and preset list
