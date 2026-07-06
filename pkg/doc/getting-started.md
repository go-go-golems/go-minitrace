---
Title: Getting Started
Slug: getting-started
Short: Step-by-step tutorial from installation through your first query
Topics:
- minitrace
- tutorial
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: Tutorial
---

This tutorial walks through installing go-minitrace, discovering your AI agent sessions, converting them to the minitrace format, and running your first analysis queries.

## Install

With Homebrew (once released):

```bash
brew tap go-go-golems/go-go-go
brew install go-minitrace
```

From source:

```bash
go install github.com/go-go-golems/go-minitrace/cmd/go-minitrace@latest
```

Verify the installation:

```bash
go-minitrace --version
```

## Step 1: Discover your sessions

Before converting anything, see what sessions you have. Each discovery command scans a native session store and reports what it finds.

If you use Claude Code:

```bash
go-minitrace discover claude-code
```

If you use Codex:

```bash
go-minitrace discover codex --source-dir ~/.codex
```

If you use Pi:

```bash
go-minitrace discover pi
```

If you use the GitHub Copilot CLI:

```bash
go-minitrace discover copilot
```

Each command outputs a table with one row per session, showing the session ID, format hint, source path, working directory, and start time. To get a machine-readable count:

```bash
go-minitrace discover claude-code --output json | jq length
```

To narrow discovery to one project or time window, use the shared filters:

```bash
go-minitrace discover pi --cwd-contains my-repo --since 2026-06-01
```

The other source formats (claude.ai, ChatGPT, turnsdb) do not have discover commands because they take explicit file paths rather than scanning a directory tree.

## Step 2: Convert your sessions

Pick one of the conversion commands and point it at your session store. The converter reads the native format and writes `.minitrace.json` files to the output directory.

Convert Claude Code sessions:

```bash
go-minitrace convert claude-code --output-dir ./output
```

This reads from `~/.claude/projects/` by default and writes to `./output/active/YYYY-MM/<id>.minitrace.json`. A `manifest.json` file is written alongside with session metadata.

To preview what would be converted without writing anything:

```bash
go-minitrace convert claude-code --dry-run
```

You can convert multiple source formats into the same output directory. Each session gets its own file, and the manifest tracks all of them:

```bash
go-minitrace convert pi --output-dir ./output
go-minitrace convert turnsdb --source /tmp/turns.db --output-dir ./output
```

To convert only specific sessions (for example the ones a filtered `discover` run surfaced), pass their paths directly with the repeatable `--source-session` flag or a `--source-list` file — no staging directory needed:

```bash
go-minitrace convert pi \
  --source-session ~/.pi/agent/sessions/.../session.jsonl \
  --output-dir ./output
```

## Step 3: Run your first query

The query command builds a normalized SQLite database from your converted archive (cached between runs) and runs analysis against it. Start with the built-in presets:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset session-list
```

This shows every session with its framework, model, turn count, tool count, duration, and read ratio.

Try the framework summary to see aggregate statistics:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset framework-summary
```

## Step 4: Explore with presets

Nine built-in presets are available:

| Preset | What it shows |
|--------|---------------|
| `session-list` | One row per session with key metadata |
| `framework-summary` | Aggregate stats grouped by agent framework |
| `tool-operation-breakdown` | Tool call counts by framework and operation type |
| `tool-failures` | Failed tool calls with target and error detail |
| `timing-analysis` | Duration, active time, TTFA, and idle ratio by framework |
| `read-ratio-distribution` | Per-session read/write/execute breakdown |
| `file-operations` | Every file touch in session order |
| `file-timeline` | Chronological operations on files with result labels |
| `annotations` | All annotations across sessions |

To get JSON output for piping to other tools:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --preset framework-summary \
  --output json
```

## Step 5: Write a custom query

Once you are comfortable with the presets, write your own SQL. The converted sessions are loaded into normalized tables — `sessions`, `turns`, `tool_calls`, `annotations`, `metrics`, and more — with the interesting fields as real columns, so most queries need no JSON extraction.

Count your sessions:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT COUNT(*) AS total_sessions FROM sessions"
```

See which models you use most:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT model, COUNT(*) AS sessions
         FROM sessions
         GROUP BY model
         ORDER BY sessions DESC"
```

Find your most tool-heavy sessions:

```bash
go-minitrace query run \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT session_id, title, tool_call_count AS tools, turn_count AS turns
         FROM sessions
         ORDER BY tools DESC
         LIMIT 10"
```

## Step 6: Validate your archive

Check that your converted files are valid JSON:

```bash
go-minitrace validate --path ./output --recursive
```

This walks the directory tree, checks each JSON file, and reports any parse errors.

## Step 7: Add your first annotation

If you want to enrich the archive with human review notes, start with the annotation CLI. Annotations are first written to `./output/annotations.db`, then synced back into `.minitrace.json` when you decide to persist them into the archive.

Add a simple session-level note:

```bash
go-minitrace annotate add \
  --output-dir ./output \
  --session <SESSION_ID> \
  --category observation \
  --title "Interesting session to review later"
```

Write those annotations back into the archive:

```bash
go-minitrace annotate sync --output-dir ./output
```

If you plan to use annotations regularly, the dedicated operator guide is:

```bash
go-minitrace help annotation-playbook
```

## Next steps

- `go-minitrace help analysis-guide` — comprehensive end-to-end workflow: discover, convert, query, annotate, serve
- `go-minitrace help js-api-reference` — JavaScript runtime API for `.js` command handlers
- `go-minitrace help structured-query-commands` — author reusable `.sql` and `.js` query commands with typed flags
- `go-minitrace help query-recipes` — ready-to-use queries for common questions
- `go-minitrace help writing-queries` — learn SQL patterns for the normalized schema
- `go-minitrace help query-commands` — preset list, `--sql-file`, and query flags reference
- `go-minitrace help annotation-playbook` — step-by-step workflow for adding, syncing, and querying annotations correctly
- `go-minitrace help minitrace-schema` — understand every field in a session
- `go-minitrace help convert-commands` — detailed reference for each conversion subcommand
- `go-minitrace help troubleshooting` — solutions for common errors

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `discover` finds 0 sessions | Check that the source directory exists and contains session files |
| `convert` errors on a specific session | Use `--dry-run` first, then convert with an ID filter if available |
| `query run` returns no rows | Verify the `--archive-glob` pattern matches your output directory |
| Output is too wide for terminal | Use `--output json` and pipe to `jq` |
