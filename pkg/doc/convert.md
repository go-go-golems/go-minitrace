---
Title: Convert Commands
Slug: convert-commands
Short: Convert sessions from Claude Code, Codex, Pi, claude.ai, ChatGPT, and turnsdb into minitrace archives
Topics:
- minitrace
- convert
Commands:
- convert
- convert claude-code
- convert codex
- convert pi
- convert claude-ai
- convert chatgpt
- convert chatgpt-json
- convert turnsdb
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `convert` group reads native AI agent session stores and writes minitrace JSON archives. Each subcommand handles one source format.

All convert subcommands share these common behaviors:

- **Output directory**: `--output-dir` defaults to `./output`. Sessions are written to `output/active/YYYY-MM/<id>.minitrace.json`.
- **Manifest**: A `manifest.json` is written at the output root and in each period directory with session metadata and statistics.
- **Dry run**: `--dry-run` inspects sources and reports what would be converted without writing files.
- **Quality grading**: Every session is assigned quality A, B, or C based on content richness.
- **Path normalization**: Home directory paths in tool call inputs are normalized to `~`.
- **Content truncation**: Tool call outputs larger than 10 KB are truncated with a SHA-256 hash of the original.

## convert claude-code

Converts Claude Code sessions from the local project store.

```bash
go-minitrace convert claude-code --source-dir ~/.claude/projects --output-dir ./output
go-minitrace convert claude-code --dry-run
```

| Flag | Default | Description |
|------|---------|-------------|
| `--source-dir` | `~/.claude/projects` | Claude Code projects directory |
| `--output-dir` | `./output` | Target archive directory |
| `--dry-run` | `false` | Preview without writing |

**What it reads**: JSONL v2 transcripts, dir-v1 tool-results sessions, and subagent transcript files found in `subagents/` subdirectories.

**Subagent handling**: When a session contains subagent transcripts, each subagent becomes its own minitrace session with `source_format: claude-code-jsonl-v2+subagent`. The parent session's `Agent` tool call carries a `spawned_agent` field linking to the subagent's session ID.

**Source format values**: `claude-code-jsonl-v2` for main sessions, `claude-code-jsonl-v2+subagent` for subagent sessions.

## convert codex

Converts Codex sessions from the local session store.

```bash
go-minitrace convert codex --source-dir ~/.codex --output-dir ./output
```

| Flag | Default | Description |
|------|---------|-------------|
| `--source-dir` | `~/.codex` | Codex base directory |
| `--output-dir` | `./output` | Target archive directory |
| `--dry-run` | `false` | Preview without writing |

**What it reads**: Session JSONL files from `~/.codex/sessions/` and exec JSONL files from `codex exec --json` output.

**Known limitation**: Older Codex sessions with unrecognized JSONL formats produce an `unsupported Codex format hint: unknown-jsonl` error. These sessions are skipped by the discovery phase but cause a hard error if the converter encounters them.

**Source format value**: `codex-session-jsonl-v1` or similar depending on the detected format.

## convert pi

Converts Pi agent sessions from the local session store.

```bash
go-minitrace convert pi --source-dir ~/.pi/agent/sessions --output-dir ./output
go-minitrace convert pi --source-session /path/to/session.jsonl --output-dir ./output
```

| Flag | Default | Description |
|------|---------|-------------|
| `--source-dir` | `~/.pi/agent/sessions` | Pi sessions directory |
| `--source-session` | | Single JSONL file to convert |
| `--output-dir` | `./output` | Target archive directory |
| `--dry-run` | `false` | Preview without writing |

**What it reads**: JSONL v3 session files. The directory structure uses workspace-encoded paths like `--home-manuel-code-foo--/`.

**Source format value**: `pi-agent-jsonl-v3`.

## convert claude-ai

Converts a claude.ai privacy data export into minitrace sessions.

```bash
go-minitrace convert claude-ai --source ~/Downloads/data-2026-03-29-11-53-11-batch-0000.zip --output-dir ./output
go-minitrace convert claude-ai --source ~/Downloads/export.zip --uuid-filter abc123,def456
```

| Flag | Default | Description |
|------|---------|-------------|
| `--source` | | Path to claude.ai data export ZIP (required) |
| `--output-dir` | `./output` | Target archive directory |
| `--uuid-filter` | | Comma-separated conversation UUID prefixes to include |
| `--dry-run` | `false` | Preview without writing |

**How to get the export**: On claude.ai, go to Settings → Privacy → Export data. You will receive a ZIP file by email.

**What it reads**: The ZIP contains a JSON file with all conversations. Each conversation becomes one minitrace session.

**Source format value**: `claude-ai-privacy-export-v1`.

## convert chatgpt

Converts a ChatGPT data export ZIP into minitrace sessions.

```bash
go-minitrace convert chatgpt --source ~/Downloads/chatgpt-export.zip --output-dir ./output
go-minitrace convert chatgpt --source ~/Downloads/export.zip --id-filter 69c7,69c8 --dry-run
```

| Flag | Default | Description |
|------|---------|-------------|
| `--source` | | Path to ChatGPT export ZIP (required) |
| `--output-dir` | `./output` | Target archive directory |
| `--id-filter` | | Comma-separated conversation ID prefixes |
| `--dry-run` | `false` | Preview without writing |

**How to get the export**: On ChatGPT, go to Settings → Data controls → Export data. You will receive a download link by email.

**What it reads**: The ZIP contains `conversations.json` with all conversations. Message parts are assembled into turns.

**Source format value**: `chatgpt-export-zip-v1`.

## convert chatgpt-json

Converts per-conversation ChatGPT JSON transcript files. This is a different, richer format than the standard account export.

```bash
go-minitrace convert chatgpt-json --source-dir /tmp/chatgpt-exports --output-dir ./output
go-minitrace convert chatgpt-json --source-dir /tmp/exports --id-filter 69c7 --dry-run --output json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--source-dir` | | Directory containing one JSON file per conversation (required) |
| `--output-dir` | `./output` | Target archive directory |
| `--id-filter` | | Comma-separated conversation ID prefixes |
| `--dry-run` | `false` | Preview without writing |

**What it reads**: Each JSON file represents one conversation. This format includes tool call details that the standard export ZIP does not.

**Source format value**: `chatgpt-json-transcript-v1`.

## convert turnsdb

Converts a Geppetto/Pinocchio `turns.db` SQLite database into minitrace sessions.

```bash
go-minitrace convert turnsdb --source /tmp/turns.db --output-dir ./output
go-minitrace convert turnsdb --source /tmp/turns.db --conv-id 5cf06c5f-0460-485e-a7c5-92d56af826f9 --dry-run --output json
```

| Flag | Default | Description |
|------|---------|-------------|
| `--source` | | Path to the turns.db SQLite file (required) |
| `--output-dir` | `./output` | Target archive directory |
| `--conv-id` | | Convert only this conversation ID |
| `--dry-run` | `false` | Preview without writing |

**What it reads**: The SQLite database stores conversation snapshots. The converter reconstructs logical conversation deltas from canonical turn snapshots using a diffing algorithm, rather than treating each stored row as a separate transcript turn.

**Source format value**: `pinocchio-turns-sqlite-v1`.

## Output directory structure

After conversion, the output directory looks like this:

```
output/
├── manifest.json                       # Root manifest with all sessions
└── active/
    ├── 2026-02/
    │   ├── <id>.minitrace.json        # One file per session
    │   ├── <id>.minitrace.json
    │   └── manifest.json              # Period manifest
    └── 2026-03/
        ├── <id>.minitrace.json
        └── manifest.json
```

Sessions are bucketed by the month they started. The root `manifest.json` contains aggregate statistics (total sessions, breakdowns by profile/quality/classification, date range).

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `unsupported Codex format hint: unknown-jsonl` | Older Codex session with unrecognized format | Known limitation; these sessions cannot be converted yet |
| Empty output after conversion | Source directory does not contain expected file patterns | Run `discover` first to verify sessions exist at the expected path |
| Very large number of output files | Claude Code subagents each become separate sessions | This is expected; filter subagents in queries using `provenance->>'source_format'` |
| Permission denied reading source | Session files owned by another user | Check file ownership and permissions |

## See also

- `go-minitrace help what-is-minitrace` — conceptual overview of the pipeline
- `go-minitrace help discover-commands` — inspect sources before converting
- `go-minitrace help minitrace-schema` — understand the output format
- `go-minitrace help adapter-reference` — detailed per-adapter conversion behavior
