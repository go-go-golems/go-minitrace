---
Title: Discover Commands
Slug: discover-commands
Short: Scan native session stores to see what sessions are available before converting
Topics:
- minitrace
- discover
Commands:
- discover
- discover claude-code
- discover codex
- discover pi
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `discover` group scans native AI agent session stores and reports what sessions are available. Discovery does not write any files or perform conversion — it is a read-only inspection tool.

Use discovery to answer questions like:
- How many sessions do I have?
- What format are my sessions in?
- Where on disk are they stored?
- Is my source directory configured correctly before I run a conversion?

## Output format

Each discover command emits one row per detected session with these fields:

| Field | Description |
|-------|-------------|
| `id` | Session identifier from the source format |
| `format_hint` | Detected format type (e.g., `jsonl-v2`, `jsonl-v3`) |
| `source_path` | Absolute path to the session file on disk |

The output goes through Glazed, so you can choose your format:

```bash
go-minitrace discover claude-code --output json
go-minitrace discover claude-code --output yaml
go-minitrace discover claude-code --output csv
```

## discover claude-code

Scans the Claude Code project directory for session files.

```bash
go-minitrace discover claude-code
go-minitrace discover claude-code --source-dir ~/.claude/projects
go-minitrace discover claude-code --output json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--source-dir` | `~/.claude/projects` | Directory to scan |

This finds JSONL v2 transcripts and dir-v1 tool-results sessions. Subagent files within `subagents/` subdirectories are discovered as well.

## discover codex

Scans the Codex directory for session and exec JSONL files.

```bash
go-minitrace discover codex --source-dir ~/.codex
go-minitrace discover codex --source-dir ~/.codex --output json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--source-dir` | `~/.codex` | Directory to scan |

## discover pi

Scans the Pi agent sessions directory for JSONL session files.

```bash
go-minitrace discover pi
go-minitrace discover pi --source-dir ~/.pi/agent/sessions
go-minitrace discover pi --output json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--source-dir` | `~/.pi/agent/sessions` | Directory to scan |

Pi session directories use workspace-encoded path names like `--home-manuel-code-foo--/` containing timestamped JSONL files.

## Why some formats lack discover

The `claude-ai`, `chatgpt`, `chatgpt-json`, and `turnsdb` converters do not have discover subcommands. These formats take explicit file paths (`--source` for ZIPs and SQLite files, `--source-dir` for JSON directories) rather than scanning a directory tree, so there is no discovery phase — you already know the file you want to convert.

## Common patterns

Count sessions across all sources:

```bash
echo "Claude Code: $(go-minitrace discover claude-code --output json | jq length)"
echo "Codex:       $(go-minitrace discover codex --source-dir ~/.codex --output json | jq length)"
echo "Pi:          $(go-minitrace discover pi --output json | jq length)"
```

List unique format hints:

```bash
go-minitrace discover claude-code --output json | jq '[.[].format_hint] | unique'
```

Find sessions in a specific project directory:

```bash
go-minitrace discover claude-code --output json \
  | jq '[.[] | select(.source_path | contains("my-project"))]'
```

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| Discover finds 0 sessions | Source directory does not exist or is empty | Check that the path exists: `ls ~/.claude/projects/` |
| Discover reports unexpected format hints | Source files are from a newer or older version | Update go-minitrace or check if the format is supported |
| Discover is slow | Very large session directory with thousands of files | This is expected; discovery walks the full tree |

## See also

- `go-minitrace help convert-commands` — convert discovered sessions into minitrace archives
- `go-minitrace help what-is-minitrace` — overview of the discover → convert → query pipeline
