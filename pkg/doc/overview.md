---
Title: go-minitrace Overview
Slug: go-minitrace-overview
Short: Overview of the Go port bootstrap and current command surface
Topics:
- minitrace
- go
- glazed
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

# go-minitrace Overview

`go-minitrace` is the new Go repository for the minitrace port.

The current implementation focuses on:

- a Glazed-based root CLI,
- Claude Code discovery and conversion, including subagents,
- Codex discovery and conversion,
- Pi local session conversion,
- claude.ai export conversion,
- ChatGPT ZIP export conversion,
- alternate ChatGPT JSON transcript conversion with extracted tool calls,
- Geppetto/Pinocchio `turns.db` snapshot conversion,
- DuckDB query recipes for converted archives,
- basic JSON validation while the full schema validator is ported.

The long-term source of truth for format semantics remains the Python/reference repo until the Go validator and converters reach parity.
