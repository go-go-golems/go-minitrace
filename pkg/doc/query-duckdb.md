---
Title: Query DuckDB
Slug: query-duckdb
Short: Query converted minitrace archives with built-in DuckDB support
Topics:
- minitrace
- duckdb
- glazed
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Example
---

# Query DuckDB

`go-minitrace query duckdb` loads minitrace JSON archives into DuckDB and runs either a named preset or ad hoc SQL.

Examples:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset session-list

go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset framework-summary \
  --output json

go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql 'SELECT COUNT(*) AS sessions FROM sessions_base'
```

Built-in presets:

- `session-list`
- `framework-summary`
- `tool-operation-breakdown`
- `timing-analysis`
- `read-ratio-distribution`
- `annotations`

The command uses explicit Glazed fields and sections for all configuration. It does not rely on hidden environment variables.
