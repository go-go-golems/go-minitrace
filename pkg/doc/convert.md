---
Title: Convert Commands
Slug: convert-commands
Short: Convert Claude Code and Codex sessions into minitrace archives
Topics:
- minitrace
- convert
- claude-code
- codex
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: GeneralTopic
---

# Convert Commands

The `convert` group writes minitrace session archives and manifests.

Examples:

```bash
go-minitrace convert claude-code --source-dir ~/.claude/projects --output-dir ./output
go-minitrace convert codex --source-dir ~/.codex --dry-run --output json
```

Current Claude Code coverage:

- JSONL v2 transcripts
- dir-v1 tool-results sessions
- subagent transcripts with parent-session backlinking

Current Codex coverage:

- session JSONL from `~/.codex/sessions/`
- exec JSONL from `codex exec --json`
