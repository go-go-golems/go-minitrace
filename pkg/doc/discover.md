---
Title: Discover Commands
Slug: discover-commands
Short: Inspect Claude Code and Codex session stores before conversion
Topics:
- minitrace
- discover
- claude-code
- codex
IsTemplate: false
IsTopLevel: false
ShowPerDefault: true
SectionType: Tutorial
---

# Discover Commands

Use discovery commands to inspect native session stores without writing minitrace output.

Examples:

```bash
go-minitrace discover claude-code --source-dir ~/.claude/projects
go-minitrace discover codex --source-dir ~/.codex --output yaml
```

The current implementation emits one row per discovered session-like source.
