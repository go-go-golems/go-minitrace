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
.read queries/overview/session-list.sql
.read queries/overview/framework-summary.sql
.read queries/tools/tool-operation-breakdown.sql
```

## One-shot usage

You can also run a query file after loading the temp table:

```bash
duckdb analysis.duckdb -init queries/load.sql -f queries/overview/session-list.sql
```

## Available files

### `load.sql` — bootstrap

| File | Purpose |
|------|---------|
| `load.sql` | Load the archive into a temp table once per DuckDB session |

### `overview/` — session-level summaries

| File | Purpose |
|------|---------|
| `session-list.sql` | List sessions with key metadata |
| `framework-summary.sql` | Aggregate stats by framework |
| `annotations.sql` | Query annotations from the attached annotations DB |

### `tools/` — tool call analysis

| File | Purpose |
|------|---------|
| `tool-operation-breakdown.sql` | Count tool calls by framework and operation type |
| `tool-failures.sql` | All failed tool calls with error details |
| `read-ratio-distribution.sql` | Inspect read-before-write style behavior |
| `exit-codes.sql` | Tool calls with explicit process exit codes |
| `justifications.sql` | Tool calls that include an explicit justification |

### `timing/` — temporal analysis

| File | Purpose |
|------|---------|
| `timing-analysis.sql` | Compare duration, TTFA, and idle ratio |

### `files/` — file-level operation tracking

| File | Purpose |
|------|---------|
| `file-operations.sql` | Every read/write/edit on files in turn order |
| `file-timeline.sql` | Chronological ops on files matching a path pattern (edit the `LIKE '%'` filter at the bottom) |

### `framework-metadata/` — framework-specific metadata examples

| File | Purpose |
|------|---------|
| `codex-exec-metadata.sql` | Inspect preserved Codex execution metadata such as source, parsed command info, stdout/stderr, and exit code |
| `claude-turn-metadata.sql` | Inspect preserved Claude Code turn metadata such as entrypoint, thread context, stop reason, and cache buckets |
| `pi-edit-diffs.sql` | Inspect preserved Pi edit diffs and first changed line metadata |

## Custom queries

Drop your own queries in `my-queries/`. They won't be tracked by git (it's in `.gitignore`).

## Preset names for CLI usage

Only the built-in preset set is available via `go-minitrace query duckdb --preset <name>`.
The example queries in `queries/` are plain SQL files on disk and are **not** compiled into the binary.

Current built-in presets:

```
annotations, framework-summary, session-list,
tool-operation-breakdown, tool-failures, read-ratio-distribution,
timing-analysis,
file-operations, file-timeline
```

## Why this shape

The original Python-side workflow mostly runs `read_json(...)` directly inside each query. That is fine for occasional one-off analysis, but if you run several queries in sequence it repeats the JSON scan each time.

This folder keeps DuckDB's schema-on-read model, but makes repeated querying cheaper within one session by materializing a temp table first.
