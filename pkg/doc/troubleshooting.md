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

**Solution**: The converter skips sessions that fail to convert and keeps going. Each failed session is reported as a diagnostics row with `status: failed` and an `error` column describing what went wrong; successfully converted sessions are written normally. The command exits with an error only if every discovered session failed to convert. To find the problematic sessions, filter the output rows:

```bash
go-minitrace convert codex --source-dir ~/.codex --output-dir ./output --output json | jq '.[] | select(.status == "failed")'
```

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
WHERE source_format NOT LIKE '%subagent%'
```

## Query errors

### query references disallowed table/view "sqlite_master"

**What happened**: A query against `sqlite_master` (or any other object outside the allowlist) was rejected with:

```
query references disallowed table/view "sqlite_master"; use db.schema() or db.tables() from JS to introspect the schema
```

**Cause**: All SQL runs through a sandboxed read-only runner that only allows the normalized tables (`sessions`, `turns`, `tool_calls`, `turn_tool_calls`, `files`, `annotations`, `handovers`, `metrics`, `attachments`, `events`) plus the `sessions_base` compatibility view. Introspection tables are blocked.

**Solution**: Use `db.schema()` or `db.tables()` from a JS handler, or `go-minitrace help minitrace-schema` for the column reference.

### no such function: UNNEST (or other DuckDB-era SQL errors)

**What happened**: Saved SQL written for the removed DuckDB engine fails with missing functions (`UNNEST`, `LEFT`, `json_extract_string`) or unknown columns.

**Cause**: The DuckDB backend was removed. Queries now run on normalized SQLite: what used to be JSON arrays inside `sessions_base` are real child tables.

**Solution**: Rewrite against the normalized tables. `UNNEST(tool_calls)` becomes `FROM tool_calls`, `LEFT(x, n)` becomes `substr(x, 1, n)`, `CAST(x AS DATE)` becomes `date(x)`. See `go-minitrace help query-duckdb` for the full migration table. Session-level `->>` SQL still works against the `sessions_base` compatibility view.

### Warning: --db-path, --table-name, and --persist-loaded are deprecated

**What happened**: Running a SQL query command printed:

```
Warning: --db-path, --table-name, and --persist-loaded are deprecated; SQL query commands now run against the normalized SQLite database built from --archive-glob.
```

**Cause**: These flags configured the removed DuckDB backend. SQL commands ignore them; JS commands still see the values on `mt.runtime` for backwards compatibility, but they no longer affect where queries run.

**Solution**: Drop the flags. Point `--archive-glob` at your archives; the normalized database is built (and cached) automatically. In SQL command files, `{{TABLE_NAME}}` now renders as the `sessions_base` compatibility view.

### Accessing bash command text in tool_calls

**What happened**: You want to query the shell command from bash tool calls.

**Cause**: In the old engine this required guessing adapter-specific JSON paths. The normalized schema promotes it to a real column.

**Solution**: Use the `command` column directly:

```sql
SELECT session_id, tool_name, command
FROM tool_calls
WHERE command LIKE '%docmgr%';
```

Raw adapter-specific argument payloads remain available in `arguments_json` and `raw_json` when you need them:

```sql
SELECT json_extract(arguments_json, '$.command') AS raw_cmd
FROM tool_calls
WHERE tool_name = 'bash'
LIMIT 5;
```

### Query returns 0 rows

**What happened**: A query ran successfully but returned no results.

**Cause**: The `--archive-glob` pattern does not match any files, or the WHERE clause filters everything out.

**Solution**: First check what loaded:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql 'SELECT COUNT(*) AS sessions FROM sessions'
```

If this returns 0, check the glob path with `ls`:

```bash
ls ./output/active/*/*.minitrace.json | head
```

### one of preset, sql, or sql-file must be specified

**What happened**: The query command exited with this error.

**Cause**: No query mode flag was given.

**Solution**: Add one of the three query mode flags:

```bash
go-minitrace query run --archive-glob '...' --preset session-list
```

### preset, sql, and sql-file are mutually exclusive

**What happened**: The command rejected the flag combination.

**Cause**: More than one query mode was specified.

**Solution**: Use exactly one of `--preset`, `--sql`, or `--sql-file`.

### JS command failed: compact error and JSON envelope

**What happened**: A JavaScript query command failed and printed a one-line error like:

```
Error: no sessions matched (file.js:12:3)
```

and, when run with `--output json`, a JSON object like:

```json
{"error":"Error: no sessions matched","location":"file.js:12:3","command":"my-command"}
```

**Cause**: This is the intended failure shape. JS errors are collapsed to the first error line plus the first stack-frame location instead of a full native stack trace, and `--output json` emits a parseable `{"error": ...}` envelope on stdout so JSON consumers never receive empty output.

**Solution**: Read the `error` and `location` fields; fix the script at the reported file/line. Automation consuming `--output json` should treat the presence of an `error` key as failure.

### Child tables return no rows

**What happened**: A query over `tool_calls` or `turns` produces no output.

**Cause**: All matched sessions have no tool calls/turns, or your WHERE clause filters out all rows.

**Solution**: Check that your filter includes sessions that actually have data:

```sql
SELECT session_id, tool_call_count AS tools
FROM sessions
WHERE tool_call_count > 0
LIMIT 5;
```

Then verify the raw distribution without the extra WHERE clause:

```sql
SELECT tool_name AS tool, COUNT(*) AS uses
FROM tool_calls
GROUP BY tool
ORDER BY uses DESC;
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

**Solution**: Convert in batches, or convert only the sessions you need with `--source-session`/`--source-list` (pi, codex, claude-code) after narrowing candidates with `discover ... --cwd-contains ... --since ...`.

### Queries are slow on large archives

**What happened**: The first query over a large archive set takes a long time.

**Cause**: The normalized SQLite database is built from the matching `.minitrace.json` files on first use.

**Solution**: The build is cached by archive fingerprint, so subsequent queries against the same globs reuse the cached database and are fast. If the first build times out, raise `--timeout-ms`, or narrow `--archive-glob` to the sessions you actually need.

## See also

- `go-minitrace help getting-started` — tutorial covering the basic workflow
- `go-minitrace help convert-commands` — conversion flags and per-adapter details
- `go-minitrace help query-commands` — query modes and flags
