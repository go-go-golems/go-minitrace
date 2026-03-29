# DuckDB Queries for go-minitrace

Query converted minitrace archives directly with DuckDB.

The recommended workflow for multiple queries in sequence is:

1. Open one DuckDB session.
2. Load the archive once with `queries/load.sql`.
3. Run as many query files as you want against the in-memory temp table.

That avoids repeating `read_json(...)` for every single query.

## Default archive path

The SQL files default to:

```sql
'./output/active/*/*.minitrace.json'
```

If your archive lives somewhere else, edit `queries/load.sql` before running it.

## Recommended interactive workflow

```bash
duckdb analysis.duckdb
```

Then inside DuckDB:

```sql
.read queries/load.sql
.read queries/session-list.sql
.read queries/framework-summary.sql
.read queries/tool-operation-breakdown.sql
```

## One-shot usage

You can also run a query file after loading the temp table:

```bash
duckdb analysis.duckdb -init queries/load.sql -f queries/session-list.sql
```

## Available files

| File | Purpose |
|------|---------|
| `load.sql` | Load the archive into a temp table once per DuckDB session |
| `session-list.sql` | List sessions with key metadata |
| `framework-summary.sql` | Aggregate stats by framework |
| `tool-operation-breakdown.sql` | Count tool calls by framework and operation type |
| `timing-analysis.sql` | Compare duration, TTFA, and idle ratio |
| `read-ratio-distribution.sql` | Inspect read-before-write style behavior |
| `annotations.sql` | Unnest annotations across sessions |

## Why this shape

The original Python-side workflow mostly runs `read_json(...)` directly inside each query. That is fine for occasional one-off analysis, but if you run several queries in sequence it repeats the JSON scan each time.

This folder keeps DuckDB's schema-on-read model, but makes repeated querying cheaper within one session by materializing a temp table first.
