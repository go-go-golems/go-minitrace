---
Title: Query Commands
Slug: query-commands
Short: Query converted minitrace archives with DuckDB using presets, custom SQL, or structured commands
Topics:
- minitrace
- duckdb
- glazed
- sqleton
Commands:
- query
- query duckdb
- query commands
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `query` group loads converted minitrace archives into an analysis backend and runs queries against them. Today that backend is DuckDB, but there are now two distinct user-facing workflows on top of it:

- `query duckdb` for raw presets, inline SQL, and SQL files
- `query commands` for repository-backed sqleton-style structured commands

Both workflows ultimately execute read-only SQL against the same loaded DuckDB table. The difference is where the SQL comes from: `query duckdb` takes it directly from a preset, string, or file, while `query commands` renders it from a structured command definition with typed parameters.

## Query workflow choices

Choose `query duckdb` when you are exploring, prototyping, or already have raw SQL in hand. It is the shortest path from idea to result.

Choose `query commands` when the query should become a named, reusable analysis tool. Structured commands give you typed parameters, aliases, CLI discoverability, and a matching form in the `/query` web UI.

For the full structured-command authoring and repository-loading guide, see `go-minitrace help structured-query-commands`.

## How `query duckdb` works

When you run `query duckdb`, go-minitrace:

1. Opens a DuckDB connection (in-memory by default, or at a specified database path)
2. Loads minitrace JSON files matching the archive glob into a table called `sessions_base`
3. Runs either a named preset, inline SQL, or a SQL file against that table
4. Streams results through Glazed for output formatting

The loading step uses DuckDB's `read_json()` with an explicit column schema and `ignore_errors = true`, so malformed files are silently skipped rather than crashing the query.

## query duckdb

```bash
go-minitrace query duckdb [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--archive-glob` | `./output/active/*/*.minitrace.json` | Glob pattern matching minitrace session files |
| `--db-path` | `:memory:` | DuckDB database path. Use a file path for persistence across runs |
| `--table-name` | `sessions_base` | Name of the table created from the loaded archive |
| `--preset` | | Named built-in query to run |
| `--sql` | | Inline SQL to run after loading |
| `--sql-file` | | Path to a SQL file to execute after loading |
| `--load-only` | `false` | Load the archive and emit a summary row without running a query |
| `--persist-loaded` | `false` | Create a persistent table instead of a temporary one |

Exactly one of `--preset`, `--sql`, `--sql-file`, or `--load-only` must be specified. They are mutually exclusive.

### Query modes

**Preset mode** runs one of the built-in queries:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset session-list
```

**Inline SQL mode** runs arbitrary SQL against the loaded archive:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT COUNT(*) AS total FROM sessions_base"
```

**SQL file mode** reads a query from a file. This is useful for saved query libraries:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql-file ./my-query.sql
```

**Load-only mode** creates the table and reports what was loaded without running a query. Use this to verify loading before querying interactively:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --load-only
```

### Built-in presets

| Preset | Description |
|--------|-------------|
| `session-list` | One row per session: id, framework, model, title, turns, tools, duration, read ratio, start time, source format |
| `framework-summary` | Aggregate stats by agent framework: session count, average tools/turns/read ratio/duration/TTFA |
| `tool-operation-breakdown` | Tool call counts grouped by framework and operation type (READ, MODIFY, NEW, EXECUTE, DELEGATE) |
| `timing-analysis` | Duration, active time, TTFA, idle ratio, min/max duration by framework |
| `read-ratio-distribution` | Per-session breakdown of reads, modifies, creates, executes with read ratio |
| `annotations` | All annotations unnested: session ID, framework, annotator, category, title, scope |

### Output formatting

Query results flow through Glazed, so all standard output options work:

```bash
# Default table output
go-minitrace query duckdb --archive-glob '...' --preset session-list

# JSON for piping
go-minitrace query duckdb --archive-glob '...' --preset session-list --output json

# CSV for spreadsheets
go-minitrace query duckdb --archive-glob '...' --preset session-list --output csv

# YAML
go-minitrace query duckdb --archive-glob '...' --preset session-list --output yaml

# Select specific fields
go-minitrace query duckdb --archive-glob '...' --preset session-list --fields id,framework,turns,tools
```

### Persistent database workflows

For repeated querying, use a file-backed database with `--persist-loaded`:

```bash
# Load once
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --db-path analysis.duckdb \
  --persist-loaded \
  --load-only

# Query repeatedly without reloading
go-minitrace query duckdb \
  --db-path analysis.duckdb \
  --archive-glob '' \
  --sql "SELECT COUNT(*) FROM sessions_base"
```

Or use the external DuckDB CLI with the `queries/` directory:

```bash
duckdb analysis.duckdb -init queries/load.sql -f queries/session-list.sql
```

### The sessions_base table

The loaded table has these columns, all derived from the minitrace JSON schema:

| Column | Type | Access pattern |
|--------|------|---------------|
| `id` | VARCHAR | Direct: `id` |
| `title` | VARCHAR | Direct: `title` |
| `summary` | VARCHAR | Direct: `summary` |
| `classification` | VARCHAR | Direct: `classification` |
| `profile` | VARCHAR | Direct: `profile` |
| `provenance` | JSON | Nested: `provenance->>'source_format'` |
| `flags` | JSON | Nested: `flags->>'needs_cleaning'` |
| `environment` | JSON | Nested: `environment->>'model'` |
| `operational_context` | JSON | Nested: `operational_context->>'working_directory'` |
| `timing` | JSON | Nested: `timing->>'duration_seconds'` |
| `turns` | JSON[] | Array: `UNNEST(turns)` |
| `tool_calls` | JSON[] | Array: `UNNEST(tool_calls)` |
| `annotations` | JSON[] | Array: `UNNEST(annotations)` |
| `metrics` | JSON | Nested: `metrics->>'turn_count'` |

Use `->>'field'` to extract string values from JSON columns, then CAST to the appropriate type for numeric operations.

### Querying annotations correctly

The `annotations` column is a JSON array in the loaded archive. To work with it, unnest the array and then extract fields from each annotation object.

The most common paths are:

| Path | Meaning |
|------|---------|
| `$.annotator` | Who created the annotation |
| `$.scope.type` | `session`, `turn`, or `tool_call` |
| `$.scope.target_id` | The session ID, turn index, or tool-call ID being annotated |
| `$.content.category` | The main label such as `ai-failure` or `question` |
| `$.content.title` | Short human-readable summary |
| `$.content.detail` | Longer explanatory note |
| `$.content.tags` | Tag array |
| `$.taxonomy_mappings.minitrace` | Minitrace taxonomy codes |
| `$.taxonomy_mappings.mast` | MAST taxonomy codes |
| `$.taxonomy_mappings.toolemu` | ToolEmu taxonomy codes |
| `$.classification` | Optional classification level |

A basic annotation query looks like this:

```sql
SELECT
  id AS session_id,
  REPLACE(CAST(json_extract(ann, '$.scope.type') AS VARCHAR), '"', '') AS scope_type,
  REPLACE(CAST(json_extract(ann, '$.content.category') AS VARCHAR), '"', '') AS category,
  REPLACE(CAST(json_extract(ann, '$.content.title') AS VARCHAR), '"', '') AS title
FROM sessions_base,
     UNNEST(annotations) AS a(ann)
ORDER BY session_id;
```

One subtle but important rule: `query duckdb` reads the `.minitrace.json` archive files it loads. If you created or edited annotations through `go-minitrace annotate ...`, run `go-minitrace annotate sync --output-dir ...` first so the archive contains those changes.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| Query returns 0 rows | Archive glob doesn't match any files | Check the glob path with `ls` first |
| `no query source specified` | None of preset/sql/sql-file/load-only was given | Add `--preset`, `--sql`, `--sql-file`, or `--load-only` |
| `preset, sql, and sql-file are mutually exclusive` | More than one query mode specified | Use exactly one |
| Type error in SQL | JSON field not cast before arithmetic | Wrap in `CAST(... AS INT)` or `CAST(... AS DOUBLE)` |
| DuckDB error on ignore_errors | Very old DuckDB version | Update DuckDB; go-minitrace embeds a compatible version |

## Structured query commands

The `query commands` subgroup loads repository-backed sqleton-style command files. Those files define:

- a command name and help text
- typed Glazed fields
- optional aliases with prefilled defaults
- a SQL template that renders against `{{TABLE_NAME}}`

A simple example looks like this:

```bash
go-minitrace query commands session-list \
  --archive-glob './output/active/*/*.minitrace.json' \
  --framework codex,pi
```

The repository-backed flow is the right choice when a query should be reusable by other people, promoted into the web UI, or shared through config/env/flag-discovered command repositories.

See `go-minitrace help structured-query-commands` for:

- how repository loading works
- how to configure `queryRepositories`, `GO_MINITRACE_QUERY_REPOSITORIES`, and `--query-repository`
- how to write `/* sqleton ... */` command files
- how to write `.alias.yaml` shortcut files

## See also

- `go-minitrace help structured-query-commands` — run and author sqleton-style structured query commands
- `go-minitrace help annotation-playbook` — operator workflow for creating, syncing, and validating annotations
- `go-minitrace help writing-duckdb-queries` — how to write custom SQL against the minitrace schema
- `go-minitrace help duckdb-query-recipes` — ready-to-use query examples
- `go-minitrace help output-formats-and-pipelines` — detailed Glazed output formatting guide
- `go-minitrace help minitrace-schema` — field reference for the loaded JSON
