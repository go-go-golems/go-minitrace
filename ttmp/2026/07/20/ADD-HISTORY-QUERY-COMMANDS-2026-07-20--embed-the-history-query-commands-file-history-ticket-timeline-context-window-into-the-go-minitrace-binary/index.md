---
Title: Embed the history query commands (file-history, ticket-timeline, context-window) into the go-minitrace binary
Ticket: ADD-HISTORY-QUERY-COMMANDS-2026-07-20
Status: active
Topics:
    - query-commands
    - js
    - embedded-catalog
    - skills
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/minitracecmd/assets_test.go
      Note: test assertions for the 3 new embedded commands
    - Path: repo://pkg/minitracecmd/core/history/context-window.js
      Note: embedded context-window verb
    - Path: repo://pkg/minitracecmd/core/history/file-history.js
      Note: embedded file-history verb
    - Path: repo://pkg/minitracecmd/core/history/ticket-timeline.js
      Note: embedded ticket-timeline verb
    - Path: repo://skills/go-minitrace-transcript-analysis/SKILL.md
      Note: pointer to the new built-in history verbs
ExternalSources: []
Summary: ""
LastUpdated: 2026-07-20T14:03:46.070511772-04:00
WhatFor: ""
WhenToUse: ""
---


# Embed the history query commands (file-history, ticket-timeline, context-window) into the go-minitrace binary

## Overview

Three JS query commands (`file-history`, `ticket-timeline`, `context-window` — a claw-stuff campaign's ticket `GOGO-MINITRACE-HISTORY-VERBS-2026-07-20` originally shipped these via the `go-minitrace-transcript-analysis` skill's external `--query-repository`) are now embedded directly in the `go-minitrace` binary under `pkg/minitracecmd/core/history/`, loaded by the same `go:embed`-backed catalog as every other built-in command (`pkg/minitracecmd/assets.go`). No code changes were needed to the verbs themselves — `LoadCatalog` already treats `.js` and `.sql` sources identically regardless of whether the backing `fs.FS` is an `embed.FS` or an on-disk directory.

Done: copied the files, added `assets_test.go` assertions, full `go test ./...` + `golangci-lint` clean, `make install`'d and verified the live `~/go/bin/go-minitrace` binary runs all three verbs against real archives with **no `--query-repository` flag**, committed (`311102e`), and updated both the live skill (removed the now-redundant JS copies from all three hardlinked mirrors, rewrote the usage section) and this repo's own bundled skill copy (added a pointer).

Status: complete. Not yet pushed/PR'd upstream — local commit on branch `task/add-skill-commands` only.

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- query-commands
- js
- embedded-catalog
- skills

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
