---
Title: What is go-minitrace
Slug: what-is-minitrace
Short: Unified format and CLI for converting, querying, and analyzing AI agent sessions across frameworks
Topics:
- minitrace
- go
- glazed
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

go-minitrace converts AI agent sessions from multiple frameworks into a single structured format called **minitrace**, then lets you query and analyze them with DuckDB.

If you use Claude Code, Codex, Pi, claude.ai, ChatGPT, or Geppetto/Pinocchio, this tool reads their native session stores and produces normalized JSON archives that you can query with SQL.

It also supports a human annotation workflow layered on top of those archives: you can add session-, turn-, and tool-call-level review notes through the annotation CLI and web UI, sync them back into `.minitrace.json`, and then analyze them alongside the rest of the archive.

## The minitrace format

A minitrace session is a JSON file that captures everything about one AI agent interaction: the conversation turns, every tool call with its input and output, token usage, timing information, and computed metrics like read ratio and time to first action.

The schema is versioned (currently `minitrace-v0.2.0`) and designed for analysis rather than replay. Every session carries provenance metadata so you can trace it back to the original source.

## The three-stage pipeline

go-minitrace works in three stages:

**Discover** scans native session stores and reports what sessions are available without converting anything. Use this to count sessions, verify source directories, or preview what a conversion would process.

**Convert** reads native session files and writes minitrace JSON archives. Each session becomes one `.minitrace.json` file organized into date-bucketed directories. A manifest file tracks all converted sessions.

**Query** loads converted archives into DuckDB and runs either built-in analysis presets or custom SQL. Results flow through Glazed, so you get table, JSON, YAML, or CSV output.

```
Source stores           Minitrace archives        Analysis
~/.claude/projects  ──►  output/active/           ──►  DuckDB queries
~/.codex            ──►    2026-03/               ──►  Presets or SQL
~/.pi/agent/sessions──►      <id>.minitrace.json  ──►  JSON/CSV/table
export.zip          ──►    manifest.json
turns.db            ──►
```

## Supported source formats

| Subcommand | Source | Description |
|------------|--------|-------------|
| `convert claude-code` | `~/.claude/projects/` | JSONL v2 transcripts, dir-v1 tool-results, subagent sessions |
| `convert codex` | `~/.codex/` | Session JSONL and exec JSONL files |
| `convert pi` | `~/.pi/agent/sessions/` | JSONL v3 session files |
| `convert claude-ai` | Privacy export ZIP | Download from Settings → Privacy → Export data on claude.ai |
| `convert chatgpt` | Data export ZIP | Download from Settings → Data controls → Export data on ChatGPT |
| `convert chatgpt-json` | Per-conversation JSON | Alternate richer transcript format, one JSON file per conversation |
| `convert turnsdb` | SQLite `turns.db` | Geppetto/Pinocchio snapshot-based conversation store |

## Quick start

Discover what sessions you have:

```bash
go-minitrace discover claude-code
go-minitrace discover codex --source-dir ~/.codex
go-minitrace discover pi
```

Convert them into a minitrace archive:

```bash
go-minitrace convert claude-code --output-dir ./output
```

Query the converted archive:

```bash
go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --preset session-list
go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --preset framework-summary
```

Run a custom query:

```bash
go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "SELECT environment->>'model' AS model, COUNT(*) AS sessions
         FROM sessions_base GROUP BY model ORDER BY sessions DESC"
```

## See also

- `go-minitrace help getting-started` — step-by-step tutorial
- `go-minitrace help annotation-playbook` — how to add, sync, and query annotations correctly
- `go-minitrace help minitrace-schema` — field-by-field schema reference
- `go-minitrace help convert-commands` — all conversion subcommands
- `go-minitrace help query-commands` — query modes, presets, and custom SQL
