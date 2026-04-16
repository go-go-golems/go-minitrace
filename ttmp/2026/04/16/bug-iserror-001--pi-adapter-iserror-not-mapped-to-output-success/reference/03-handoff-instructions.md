---
Title: Handoff — go-minitrace Pi Adapter Bugs & Schema Docs
Ticket: bug-iserror-001
Status: active
Topics:
    - handoff
    - pi-adapter
    - converter
    - bug
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/adapters/pi/convert.go
      Note: Go adapter now maps message-level isError into tool call success/error state
    - Path: pkg/adapters/pi/convert_test.go
      Note: Regression coverage for successful and failed message-level tool results
    - Path: ttmp/2026/04/16/bug-iserror-001--pi-adapter-iserror-not-mapped-to-output-success/scripts/01-verify-real-session.go
      Note: Reproducible real-session verification script proving the Go fix yields 59 failures
ExternalSources: []
Summary: ""
LastUpdated: 2026-04-16T00:00:00Z
WhatFor: ""
WhenToUse: ""
---



# Handoff: Pi Adapter Bugs & Schema Documentation Gaps

## What Happened

I was setting up Jellyfin on my homelab (Proxmox + k3s). During the session, docmgr ticket files kept vanishing — written successfully, gone minutes later. I used go-minitrace to analyze the session transcript and discovered two independent problems:

1. **Pi's write/edit tools write to the real filesystem** (no sandbox, no overlay — I read the source). The files vanished for a different reason I never fully diagnosed. But the investigation is what matters — it surfaced the converter bugs.

2. **The go-minitrace Pi adapter has a bug that makes failed tool calls look successful** in the converted output. This means every analysis query filtering on `output.success` returns wrong results. The Python adapter has the same bug but worse.

## Status Update

As of 2026-04-16, the **Go adapter bug is fixed in this repo** and a **regression test was added**.

Completed:
- `pkg/adapters/pi/convert.go` now reads message-level `isError` / `is_error`
- `pkg/adapters/pi/convert_test.go` now covers:
  - successful message-level tool results
  - failed message-level tool results
  - no spurious turns created from `toolResult` messages
- `go test ./pkg/adapters/pi -count=1` passes
- `ttmp/.../scripts/01-verify-real-session.go` verifies the real Jellyfin Pi session and now reports **59 failed tool calls**

Still open:
- fix the Python adapter
- improve schema/help docs
- optionally capture additional Pi-only fields like `details.diff`

## What You Need to Know

### Tickets Created

| Ticket | Path | What It Is |
|--------|------|------------|
| `bug-iserror-001` | `ttmp/2026/04/16/bug-iserror-001--pi-adapter-iserror-not-mapped-to-output-success/` | The converter bug |
| `schema-docs-001` | `ttmp/2026/04/16/schema-docs-001--fix-minitrace-schema-and-duckdb-query-documentation/` | Documentation gaps |

### Key Files to Read (in order)

1. **`bug-iserror-001/reference/01-bug-report-iserror-not-mapped.md`** — Root cause, exact line numbers, the fix, a test fixture
2. **`bug-iserror-001/reference/02-go-vs-python-adapter-comparison.md`** — Same bug in Python adapter but worse (no tool results matched at all)
3. **`schema-docs-001/sources/minitrace-schema-doc-issues.md`** — 10 doc issues I hit writing DuckDB queries
4. **`schema-docs-001/sources/pi-transcript-vs-minitrace-field-comparison.md`** — Full field-by-field comparison of raw Pi transcript vs minitrace output

### The Bug in One Paragraph

`pkg/adapters/pi/convert.go` line 175 hardcodes `isError=false` when processing message-level toolResult objects. Pi puts ALL tool results (100% of them — 606 out of 606 in my session) as message-level objects, never inside content blocks. The content-block path that correctly reads `isError` is dead code. Result: 59 failed tool calls in my session alone all show `success=true`. Fix is one line: read `msg["isError"]` instead of hardcoding `false`.

### The Python Adapter

Same root cause but worse — it has NO message-level toolResult handling at all. Tool results never get matched to tool calls. They appear as 606 spurious "user" turns. Fix is ~15 lines.

## What to Do (Priority Order)

### 1. Fix the Python adapter (1 hour)

`/home/manuel/code/others/llms/minitrace/adapters/pi/minitrace-pi-adapter.py`

Add message-level toolResult handling after the content block loop (see `02-go-vs-python-adapter-comparison.md` for the exact code). Two things to fix:

- Match tool results to pending tool calls using `msg["toolCallId"]`
- Skip turn creation for toolResult messages (currently creates fake user turns)

### 2. Fix the schema docs (2-3 hours)

Read `schema-docs-001/sources/minitrace-schema-doc-issues.md` for 10 specific issues. The top 3:

1. `tool_name` is JSON type in DuckDB after UNNEST, not string — doc says `string`
2. No warning that you need `json_extract(tc, '$.field')` after UNNEST, not dot notation
3. No schema discovery section (DESCRIBE, json_structure)

Files to edit are in `cmd/go-minitrace/cmds/help/` — the embedded help pages for `minitrace-schema` and `writing-duckdb-queries`.

### 3. Capture lost fields (optional, 2-4 hours)

From `pi-transcript-vs-minitrace-field-comparison.md`:

- `details.diff` on edit results (unified diff — very valuable)
- `stopReason` / `errorMessage` on assistant turns
- `compaction` events with read/modified file lists

These require schema additions to `pkg/minitrace/schema.go`.

## Where the Analysis Files Are

### In bug-iserror-001/scripts/ (DuckDB queries I used)

The queries are written for the jellyfin session but are general-purpose:
- `query-jellyfin-timeline.sql` — Full timeline of file operations (most useful starting point)
- `query-all-failures.sql` — Find all failed tool calls (needs the isError fix to work correctly)
- `query-docmgr-commands.sql` — Track docmgr ticket lifecycle
- `query-git-operations.sql` — Find git operations that might affect files
- `query-deletion-operations.sql` — Find rm/clean/checkout operations
- `query-jellyfin-file-operations.sql` — Track writes/edits on specific files
- Others are exploration/scaffolding queries

### In bug-iserror-001/sources/

- `pi-transcript-vs-minitrace-field-comparison.md` — The complete field diff

### In schema-docs-001/sources/

- `minitrace-schema-doc-issues.md` — The 10 doc issues
- `pi-transcript-vs-minitrace-field-comparison.md` — Copy of the field comparison
- SQL queries duplicated from bug ticket

### In schema-docs-001/scripts/

- `reconstruct_files.py` — Tool I wrote to recover lost files from minitrace archives. Useful for anyone who hits the same issue. Replays write/edit tool calls to rebuild files.

### Source session for testing

The Pi session that triggered all this:
```
~/.pi/agent/sessions/--home-manuel-code-wesen-crib-k3s--/2026-04-16T01-34-34-242Z_2035dd97-cfb1-47ba-a90d-41096ae624d5.jsonl
```

Already converted to minitrace at (in crib-k3s repo):
```
~/code/wesen/crib-k3s/analysis/jellyfin-session/active/2026-04/2035dd97-cfb1-47ba-a90d-41096ae624d5.minitrace.json
```

## Quick Sanity Check After Fixes

### Reliable verification path used in this ticket

```bash
# Converts the real Pi session via the local adapter package, then counts failures
# Expected output includes: failed_tool_calls: 59

go run ./ttmp/2026/04/16/bug-iserror-001--pi-adapter-iserror-not-mapped-to-output-success/scripts/01-verify-real-session.go
```

### Notes on CLI-based verification

Two extra wrinkles showed up during verification:

1. The checked-in `./go-minitrace` binary in the repo was stale, so it did **not** include the adapter fix.
2. The unrelated `pkg/minitracecmd/repositories.go` compile issue has since been fixed by migrating it to Glazed's declarative config-plan API (`config.NewPlan` + `SystemAppConfig` / `HomeAppConfig` / `XDGAppConfig`).

So the ticket-local verification script above remains the most direct reproducible path, but `go run ./cmd/go-minitrace --help` now builds again.

## Things I Didn't Get To

- The underlying cause of the vanishing docmgr files (the bug that started all this). The write/edit tools write to real fs, no overlay. No rm/git commands ran. Files just disappeared. Could be OS-level (tmpfiles.d, systemd-tmpfiles?) or something in docmgr itself.
- The `details.diff` field capture — would need a schema addition
- Testing the fix against other Pi sessions to see how widespread the `isError` bug is
