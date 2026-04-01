---
Title: Query Commands
Slug: query-commands
Short: Query converted minitrace archives from the CLI
Topics:
- minitrace
- duckdb
- glazed
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Tutorial
---

# Query Commands

The `query` group is for querying converted minitrace archives directly from `go-minitrace`.

The first backend is DuckDB.

Examples:

```bash
go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --preset session-list
go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --sql 'SELECT COUNT(*) AS sessions FROM sessions_base'
```

The command is authored as a Glazed command and accepts its configuration through explicit Glazed fields and sections.
