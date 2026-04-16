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
go-minitrace query commands overview session-list --archive-glob './output/active/*/*.minitrace.json'
go-minitrace validate --path /path/to/file-or-dir --recursive
```

Query the converted archive with DuckDB:

```bash
duckdb analysis.duckdb -init queries/load.sql -f queries/session-list.sql
duckdb analysis.duckdb -init queries/load.sql -f queries/framework-summary.sql
```

## Structured query commands

In addition to `query duckdb`, go-minitrace now supports repository-backed structured query commands under:

```bash
go-minitrace query commands ...
```

Repository subdirectories become nested CLI groups. For example, `pkg/minitracecmd/core/overview/session-list.sql` is exposed as:

```bash
go-minitrace query commands overview session-list
```

These commands use sqleton-style SQL files with a `/* sqleton ... */` YAML preamble. That preamble defines the command name, help text, and typed parameters, while the SQL body remains a normal SQL template rendered against the loaded DuckDB table.

This is the right workflow when a query should be reusable, parameterized, and visible in the web UI instead of living only as an ad hoc `--sql` string.

### Quick examples

Run an embedded command:

```bash
go-minitrace query commands overview session-list \
  --archive-glob './output/active/*/*.minitrace.json'
```

Run an alias command with prefilled defaults:

```bash
go-minitrace query commands overview aliases codex-framework-summary \
  --archive-glob './output/active/*/*.minitrace.json'
```

Load additional command repositories:

```bash
go-minitrace query commands overview framework-summary \
  --query-repository ./query-commands/team \
  --archive-glob './output/active/*/*.minitrace.json'
```

External repositories follow the same nested-tree pattern. For example:

```text
query-commands/
└── overview/
    ├── session-list.sql
    └── aliases/
        └── codex-framework-summary.alias.yaml
```

becomes:

```bash
go-minitrace query commands overview session-list
go-minitrace query commands overview aliases codex-framework-summary
```

Or configure repositories globally:

```yaml
queryRepositories:
  - ./query-commands/team
```

```bash
export GO_MINITRACE_QUERY_REPOSITORIES=./query-commands/team:./query-commands/local
```

Repository precedence is:

1. `--query-repository`
2. `GO_MINITRACE_QUERY_REPOSITORIES`
3. `queryRepositories` in app config
4. embedded commands last

### Writing a command

A minimal command file looks like this:

```sql
/* sqleton
name: session-list
short: List minitrace sessions
flags:
  - name: framework
    type: stringList
    help: Filter by agent framework
  - name: limit
    type: int
    default: 100
    help: Limit the number of rows returned
*/
SELECT
  id,
  environment->>'agent_framework' AS framework,
  title
FROM {{TABLE_NAME}}
WHERE 1=1
{{ if .framework -}}
AND (environment->>'agent_framework') IN ({{ .framework | sqlStringIn }})
{{ end -}}
ORDER BY timing->>'started_at' DESC
LIMIT {{ .limit }};
```

A matching alias file looks like this:

```yaml
name: codex-framework-summary
short: Summarize only codex sessions
aliasFor: framework-summary
flags:
  framework:
    - codex
```

For the full usage and authoring guide, run:

```bash
go-minitrace help structured-query-commands
```

## Annotations

Minitrace sessions support an `annotations` array for human-authored metadata on sessions, turns, and tool calls. Annotations are stored in a parallel SQLite database and can be written via CLI or HTTP API, then synced back to `.minitrace.json` source files.

For a step-by-step operator workflow, including when to use `annotate add`, `annotate sync`, `query duckdb`, and `validate`, see:

```bash
go-minitrace help annotation-playbook
```

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

When `serve` is started with an `--archive-glob`, it automatically attaches the annotations database. The structured HTTP API now lives on `/api/v2/...`; the one deliberate JSON-native exception that remains is raw SQL execution on `POST /api/query`. Annotation endpoints return 503 if the store is unavailable.

```bash
go-minitrace serve --archive-glob './output/active/*/*.minitrace.json'
```

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v2/sessions/{id}/annotations` | List annotations for session |
| POST | `/api/v2/sessions/{id}/annotations` | Create annotation |
| GET | `/api/v2/annotations` | List all (filter: `?session=&category=&annotator=`) |
| PUT | `/api/v2/annotations/{annId}` | Patch annotation |
| DELETE | `/api/v2/annotations/{annId}` | Delete annotation |
| POST | `/api/v2/annotations/sync` | Sync SQLite → .minitrace.json |

### Validation

`go-minitrace validate` checks annotation structure (known categories, scope types, taxonomy arrays, classification levels) in addition to JSON syntax.

### DuckDB queries

There are two annotation query modes, and they behave differently:

1. **`go-minitrace serve`** attaches `outputDir/annotations.db` live via `sqlite_scanner`, so the `annotations` table is available directly.
2. **`go-minitrace query duckdb`** loads `.minitrace.json` archive files, so it sees the `annotations` arrays embedded in those files. If you used `annotate add` / `edit` / `delete` first, run `annotate sync` before expecting `query duckdb` to see the latest annotations.

With `sqlite_scanner` attached in `serve`, annotations are queryable alongside session data:

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
# Cross-session failure query against the live SQLite-backed annotations table
curl -X POST http://localhost:8080/api/query \
  -H 'Content-Type: application/json' \
  -d '{"sql":"SELECT a.session_id, a.category, a.title FROM annotations a JOIN sessions_base sb ON sb.id = a.session_id WHERE a.category = '\''ai-failure'\'''}''
```

For CLI-only querying after sync, query the embedded `annotations` arrays in the archive:

```bash
go-minitrace annotate sync --output-dir ./output

go-minitrace query duckdb \
  --archive-glob './output/active/*/*.minitrace.json' \
  --sql "
    SELECT
      id AS session_id,
      environment->>'agent_framework' AS framework,
      REPLACE(CAST(json_extract(ann, '$.content.category') AS VARCHAR), '\"', '') AS category,
      REPLACE(CAST(json_extract(ann, '$.content.title') AS VARCHAR), '\"', '') AS title
    FROM sessions_base,
         UNNEST(annotations) AS a(ann)
    WHERE REPLACE(CAST(json_extract(ann, '$.content.category') AS VARCHAR), '\"', '') = 'ai-failure'
  "
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
