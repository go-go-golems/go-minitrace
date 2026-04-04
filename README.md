# go-minitrace

Glazed-based Go port of [minitrace](https://github.com/fukami/minitrace), focused first on Claude Code, Codex, Pi, claude.ai, ChatGPT, and `turns.db` session conversion.

## What it does

- Boots a Glazed-based CLI for the Go port of minitrace
- Implements real discovery commands for Claude Code, Codex, and Pi session stores
- Converts Claude Code sessions into minitrace JSON archives, including dir-v1 tool-results sessions and subagent transcripts with parent backlinking
- Converts Codex session JSONL and exec JSONL data into minitrace JSON archives
- Converts Pi local JSONL sessions into minitrace JSON archives
- Converts claude.ai privacy export ZIPs into minitrace JSON archives
- Converts ChatGPT data export ZIPs into minitrace JSON archives
- Converts alternate per-conversation ChatGPT JSON transcript exports into minitrace JSON archives with tool-call extraction
- Converts Geppetto/Pinocchio `turns.db` snapshot stores into minitrace JSON archives via canonical snapshot diffing
- Queries converted minitrace archives through a built-in DuckDB-backed `query duckdb` command
- Ships DuckDB query recipes for converted archives under `queries/`
- Includes a basic JSON validation command while full schema validation is ported
- Keeps the repo focused on the Go implementation, separate from the Python/spec reference repo

## Install

### Homebrew

This repo is intended to be released via GoReleaser and published to the go-go-golems Homebrew tap.

```bash
brew tap go-go-golems/go-go-go
brew install go-minitrace
```

### Go install (from source)

```bash
go install github.com/go-go-golems/go-minitrace/cmd/go-minitrace@latest
```

## Quick start

```bash
go-minitrace --help
go-minitrace discover claude-code --source-dir ~/.claude/projects
go-minitrace discover codex --source-dir ~/.codex
go-minitrace discover pi --source-dir ~/.pi/agent/sessions
go-minitrace convert claude-code --source-dir ~/.claude/projects --output-dir ./output
go-minitrace convert codex --source-dir ~/.codex --output-dir ./output
go-minitrace convert pi --source-dir ~/.pi/agent/sessions --output-dir ./output
go-minitrace convert claude-ai --source ~/Downloads/data-2026-03-29-11-53-11-batch-0000.zip --output-dir ./output
go-minitrace convert chatgpt --source ~/Downloads/chatgpt-export.zip --output-dir ./output
go-minitrace convert chatgpt-json --source-dir /tmp/chatgpt-exports --output-dir ./output
go-minitrace convert turnsdb --source /tmp/turns.db --output-dir ./output
go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --preset session-list
go-minitrace validate --path /path/to/file-or-dir --recursive
```

Query the converted archive with DuckDB:

```bash
duckdb analysis.duckdb -init queries/load.sql -f queries/session-list.sql
duckdb analysis.duckdb -init queries/load.sql -f queries/framework-summary.sql
```

## Annotations

Minitrace sessions support an `annotations` array for human-authored metadata on sessions, turns, and tool calls. Annotations are stored in a parallel SQLite database and can be written via CLI or HTTP API, then synced back to `.minitrace.json` source files.

### Storage model

- **Working store**: `outputDir/annotations.db` (SQLite, WAL mode)
- **Source of truth**: `.minitrace.json` files (modified only via `annotate sync`)
- **DuckDB integration**: annotations are attached directly via DuckDB's `sqlite_scanner` extension — no export/import needed, queries are live

```
output/
├── annotations.db          ← SQLite (working store)
└── active/
    └── 2026-04/
        └── sess-001.minitrace.json  ← source of truth (canonical format)
```

### CLI

```bash
# Add annotation
go-minitrace annotate add \
  --output-dir ./output \
  --session sess-001 \
  --category ai-failure \
  --title "Authentication failure in tool call" \
  --taxonomy-minitrace F-AUT \
  --tags auth,regression

# List
go-minitrace annotate list --output-dir ./output

# Edit
go-minitrace annotate edit --id <id> --title "Updated title"

# Delete
go-minitrace annotate delete --id <id>

# Sync back to .minitrace.json
go-minitrace annotate sync --output-dir ./output
go-minitrace annotate sync --output-dir ./output --dry-run  # preview

# Import from JSON
go-minitrace annotate import --file annotations.json --output-dir ./output
```

### Annotation categories

`observation`, `ai-failure`, `user-error`, `environment-issue`, `success`, `question`, `to-discuss`, `to-improve`

### Serve + HTTP API

When `serve` is started with an `--archive-glob`, it automatically attaches the annotations database. All annotation endpoints return 503 if the store is unavailable.

```bash
go-minitrace serve --archive-glob './output/active/*/*.minitrace.json'
```

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/sessions/{id}/annotations` | List annotations for session |
| POST | `/api/sessions/{id}/annotations` | Create annotation |
| GET | `/api/annotations` | List all (filter: `?session=&category=&annotator=`) |
| PUT | `/api/annotations/{annId}` | Patch annotation |
| DELETE | `/api/annotations/{annId}` | Delete annotation |
| POST | `/api/annotations/sync` | Sync SQLite → .minitrace.json |

### Validation

`go-minitrace validate` checks annotation structure (known categories, scope types, taxonomy arrays, classification levels) in addition to JSON syntax.

### DuckDB queries

With `sqlite_scanner` attached, annotations are queryable alongside session data:

```sql
SELECT
    a.session_id,
    sb.environment->>'agent_framework' AS framework,
    a.category,
    a.title,
    a.taxonomy_m AS taxonomy_minitrace
FROM annotations a
JOIN sessions_base sb ON sb.id = a.session_id
ORDER BY a.created_at DESC;
```

Or through the serve HTTP API:

```bash
# Cross-session failure query
curl -X POST http://localhost:8080/api/query \
  -H 'Content-Type: application/json' \
  -d '{"sql":"SELECT a.session_id, a.category, a.title FROM annotations a JOIN sessions_base sb ON sb.id = a.session_id WHERE a.category = '\''ai-failure'\'''}''
```

Or query directly:

```bash
go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --sql "SELECT a.session_id, a.category, a.title FROM annotations a JOIN sessions_base sb ON sb.id = a.session_id WHERE a.category = 'ai-failure'"
```

## Or query directly through the CLI:

```bash
go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --preset framework-summary
go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --sql 'SELECT COUNT(*) AS sessions FROM sessions_base'
```

## Development

```bash
make lint
make test
make build
```

Pre-commit hooks are managed via `lefthook.yml`:

```bash
lefthook install
```

Snapshot release:

```bash
make goreleaser
```

## Security notes

Treat CLI output, logs, and exported data as sensitive until you’ve reviewed what the tool emits.

## License

MIT. See `LICENSE`.
