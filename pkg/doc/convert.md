---
Title: Convert Commands
Slug: convert-commands
Short: Planned conversion surface for Claude Code and Codex
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

The `convert` group is scaffolded first so the final CLI shape is stable early.

Examples:

```bash
go-minitrace convert claude-code --source-dir ~/.claude/projects --output-dir ./output
go-minitrace convert codex --source-dir ~/.codex --dry-run --output json
```

At the moment these commands emit planning rows based on discovery. The actual
session conversion engine is the next milestone.
