---
Title: Troubleshooting
Slug: troubleshooting
Short: Solutions for common errors and problems with go-minitrace
Topics:
- minitrace
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

This page collects solutions for common errors and unexpected behavior when using go-minitrace.

## Conversion errors

### unsupported Codex format hint: unknown-jsonl

**What happened**: The Codex converter encountered a session file with a JSONL format it does not recognize.

**Cause**: Older Codex sessions or sessions from a different Codex version may use a format the adapter has not been taught to parse.

**Solution**: This is a known limitation. The converter currently fails on the first unrecognized session rather than skipping it. There is no workaround other than removing the problematic session file from the Codex directory before conversion.

### Empty output after conversion

**What happened**: The convert command ran without errors but produced no minitrace files.

**Cause**: The source directory does not contain files matching the expected patterns, or the directory path is wrong.

**Solution**: Run `discover` first to confirm sessions exist:

```bash
go-minitrace discover claude-code --output json | jq length
```

If discover reports 0 sessions, check the `--source-dir` path.

### Permission denied reading source files

**What happened**: The converter cannot read session files.

**Cause**: Session files are owned by a different user or have restrictive permissions.

**Solution**: Check file ownership with `ls -la` on the source directory. If needed, adjust permissions or run as the appropriate user.

### Very large number of output files

**What happened**: Converting Claude Code produced many more files than expected.

**Cause**: Claude Code subagent sessions each become their own minitrace file. A session that delegates to 10 subagents produces 11 files (1 main + 10 subagent).

**Solution**: This is expected behavior. Filter subagent sessions in queries:

```sql
WHERE provenance->>'source_format' NOT LIKE '%subagent%'
```

## Query errors

### Query returns 0 rows

**What happened**: A query ran successfully but returned no results.

**Cause**: The `--archive-glob` pattern does not match any files, or the WHERE clause filters everything out.

**Solution**: First verify loading with `--load-only`:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --load-only
```

If this returns 0, check the glob path with `ls`:

```bash
ls ./output/active/*/*.minitrace.json | head
```

### no query source specified

**What happened**: The query command exited with this error.

**Cause**: None of `--preset`, `--sql`, `--sql-file`, or `--load-only` was specified.

**Solution**: Add one of the four query mode flags:

```bash
go-minitrace query duckdb --archive-glob '...' --preset session-list
```

### preset, sql, and sql-file are mutually exclusive

**What happened**: The command rejected the flag combination.

**Cause**: More than one query mode was specified.

**Solution**: Use exactly one of `--preset`, `--sql`, or `--sql-file`.

### Type errors in custom SQL

**What happened**: DuckDB reports a type error or unexpected NULL values.

**Cause**: JSON extraction with `->>` always returns strings. Doing arithmetic on a string fails.

**Solution**: Wrap in `CAST`:

```sql
-- Wrong: comparing string to number
WHERE metrics->>'turn_count' > 10

-- Right: cast first
WHERE CAST(metrics->>'turn_count' AS INT) > 10
```

### UNNEST returns no rows

**What happened**: A query using `UNNEST(tool_calls)` or `UNNEST(turns)` produces no output.

**Cause**: All matched sessions have empty arrays for the unnested column, or your WHERE clause filters out all unnested rows.

**Solution**: Check that your filter includes sessions that actually have tool calls or turns:

```sql
SELECT id, CAST(metrics->>'tool_call_count' AS INT) AS tools
FROM sessions_base
WHERE CAST(metrics->>'tool_call_count' AS INT) > 0
LIMIT 5;
```

If you are filtering on unnested tool-call fields, first verify the raw distribution without the extra WHERE clause:

```sql
SELECT tc->>'tool_name' AS tool, COUNT(*) AS uses
FROM sessions_base,
     UNNEST(tool_calls) AS t(tc)
GROUP BY tool
ORDER BY uses DESC;
```

### DuckDB JSON[] sharp edges with tool_calls

**What happened**: `tool_calls` looks present in the archive, but ad hoc SQL still returns `NULL`, odd results, or parser/type errors.

**Cause**: `tool_calls`, `turns`, and `annotations` are loaded as DuckDB `JSON[]` columns. The most common problems are:

- using `tool_calls[0]` as if lists were zero-based
- applying JSON paths to the whole `tool_calls` container instead of unnested elements
- guessing the wrong nested input field (`input.path` vs `input.file_path` vs `input.arguments.path`)
- using `->>` together with `LIKE` without parentheses

**Solution**:

1. Prefer this pattern first:

```sql
SELECT tc->>'tool_name' AS tool
FROM sessions_base,
     UNNEST(tool_calls) AS t(tc)
LIMIT 20;
```

2. Remember that DuckDB list indexing is 1-based:

```sql
SELECT
  tool_calls[1]->>'tool_name' AS first_tool,
  tool_calls[0]->>'tool_name' AS zeroth_tool
FROM sessions_base
LIMIT 1;
```

3. For file-oriented tools, use a path fallback instead of assuming `input.path` exists:

```sql
SELECT
  COALESCE(tc->'input'->>'file_path', tc->'input'->'arguments'->>'path') AS path
FROM sessions_base,
     UNNEST(tool_calls) AS t(tc)
WHERE tc->>'tool_name' IN ('read', 'write', 'edit')
LIMIT 20;
```

4. If you filter with `LIKE`, parenthesize the `->>` extraction:

```sql
WHERE (tc->'input'->>'command') LIKE '%docmgr%'
```

## Validation issues

### All files valid_json but queries fail

**What happened**: `validate` reports all files as valid JSON, but queries return unexpected results.

**Cause**: `validate` currently checks JSON syntax only, not schema compliance. Files can be valid JSON but have wrong or missing fields.

**Solution**: This is expected. Schema validation is planned for a future release. Check specific files manually if queries return unexpected data:

```bash
cat ./output/active/2026-03/abc123.minitrace.json | jq '.metrics'
```

## Discovery issues

### Discover is very slow

**What happened**: Discovery takes a long time to complete.

**Cause**: The source directory contains thousands of session files and the discovery walks the entire tree.

**Solution**: This is expected for large session stores. The discovery is read-only and will complete — it just takes time proportional to the number of files.

### Discover reports unexpected format hints

**What happened**: Discovery shows format hints you do not recognize.

**Cause**: The session files may be from a newer or older version of the source tool.

**Solution**: Check if your go-minitrace version supports the format. Update if needed:

```bash
go install github.com/go-go-golems/go-minitrace/cmd/go-minitrace@latest
```

## Performance

### Conversion is slow for large archives

**What happened**: Converting thousands of sessions takes a long time.

**Cause**: Each session file must be read, parsed, and written individually.

**Solution**: Convert in batches. For Claude Code, you can use separate source directories per project. For DuckDB queries, use `--db-path` with a file to avoid reloading on every query.

### Queries are slow on large archives

**What happened**: DuckDB queries take a long time.

**Cause**: Loading thousands of JSON files into DuckDB requires parsing each file.

**Solution**: Use a persistent database to avoid reloading:

```bash
go-minitrace query duckdb \
  --archive-glob '...' \
  --db-path analysis.duckdb \
  --persist-loaded \
  --load-only

# Now query without reloading
go-minitrace query duckdb \
  --db-path analysis.duckdb \
  --archive-glob '' \
  --sql "SELECT COUNT(*) FROM sessions_base"
```

## See also

- `go-minitrace help getting-started` — tutorial covering the basic workflow
- `go-minitrace help convert-commands` — conversion flags and per-adapter details
- `go-minitrace help query-commands` — query modes and flags
