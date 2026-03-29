# go-minitrace

Glazed-based Go port of minitrace, focused first on Claude Code, Codex, and Pi session conversion.

## What it does

- Boots a Glazed-based CLI for the Go port of minitrace
- Implements real discovery commands for Claude Code, Codex, and Pi session stores
- Converts Claude Code sessions into minitrace JSON archives, including dir-v1 tool-results sessions and subagent transcripts with parent backlinking
- Converts Codex session JSONL and exec JSONL data into minitrace JSON archives
- Converts Pi local JSONL sessions into minitrace JSON archives
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
go-minitrace validate --path /path/to/file-or-dir --recursive
```

Query the converted archive with DuckDB:

```bash
duckdb analysis.duckdb -init queries/load.sql -f queries/session-list.sql
duckdb analysis.duckdb -init queries/load.sql -f queries/framework-summary.sql
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
