# Changelog

## 2026-04-16

- Initial workspace created

## 2026-04-16

Completed intern brief schema gap analysis. Reviewed raw Codex and Claude Code transcripts, compared with converted minitrace output, identified 5 high-priority schema gaps and 5 items to keep in framework_metadata. Key findings:

1. **Exit codes flattened to boolean** - Both adapters lose rich exit codes (Codex: 0/1/2/130+, Claude: is_error bool). Cannot distinguish error types or signal-terminated processes.
2. **Justification buried in metadata** - Tool use rationale hard to query. Cannot identify operations lacking justification.
3. **Sandbox policy too coarse** - Boolean `sandbox` loses policy types (workspace-write, danger-full-access) and escalation patterns.
4. **stdout/stderr merged** - Cannot identify silent failures or analyze stderr warnings.
5. **Mode switches invisible** - Collaboration mode changes and message phases not tracked as first-class events.

Deliverables: Full analysis memo, summary table, and research diary moved to ticket sources/ directory.

### Related Files

- sources/intern-review/schema-gap-analysis.md - Comprehensive gap analysis with concrete examples
- sources/intern-review/schema-gaps-table.md - Summary table in requested format  
- sources/intern-review/active/ - Converted minitrace JSON files from analysis
- reference/intern-schema-gap-analysis-diary.md - This research diary

## 2026-04-17

Performed an independent follow-up review of the claimed Claude Code and Codex schema gaps against actual raw transcripts, current adapter behavior, and fresh local Codex conversions. Narrowed the actionable first-wave schema work to two high-confidence promotions (`exit_code`, `justification`) plus a metadata-preservation pass for richer Codex and Claude Code fields. Updated ticket tasks to reflect the validated scope instead of the broader speculative list.

### Related Files

- reference/05-validated-schema-gap-review.md - Independent review note with validated findings and recommended implementation order
- tasks.md - Updated with validated schema-gap follow-up tasks
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/codex/convert.go - Confirmed current handling of exit codes, justification, and policy metadata
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/claudecode/convert.go - Confirmed which Claude-specific fields are currently dropped
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitrace/schema.go - Confirmed missing first-class fields that block cleaner analysis

## 2026-04-16

Created ticket with full bug report, root cause analysis, and test case. Fix is one-line change in convert.go:175.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/pi/convert.go — Line 175 hardcodes isError=false

## 2026-04-16

Implemented the Go-side Pi adapter fix for message-level `toolResult.isError` handling and added a regression test that covers both successful and failed message-level tool results without creating spurious turns.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/pi/convert.go — Read `msg["isError"]`/`msg["is_error"]` before applying tool results
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/pi/convert_test.go — Added regression coverage for message-level tool results with both success and failure cases

## 2026-04-16

Verified the Go adapter fix against the real Jellyfin Pi session using a ticket-local Go script. The converter now reports 59 failed tool calls, including the previously misclassified `edit` failures with `File not found` errors.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/ttmp/2026/04/16/bug-iserror-001--pi-adapter-iserror-not-mapped-to-output-success/scripts/01-verify-real-session.go — Reproducible verification script that converts the real session and counts failed tool calls
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/pi/convert.go — Verified against the real session after the `isError` fix

## 2026-04-16

Migrated `pkg/minitracecmd/repositories.go` off the removed Glazed `ResolveAppConfigPath` helper to the new config plan API (`config.NewPlan` + `SystemAppConfig`/`HomeAppConfig`/`XDGAppConfig`), using Pinocchio's newer bootstrap style as the reference. Added overlay tests for the app-config loader and command-level tests proving `query commands` picks up repository overrides from app config, with XDG config overriding the legacy home-dotdir config.

### Related Files

- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd/repositories.go — Replaced legacy config-path resolution with Glazed config plans and layered config loading
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/minitracecmd/repositories_test.go — Added coverage for multi-file overlay behavior and omitted-field preservation
- /home/manuel/code/wesen/corporate-headquarters/go-minitrace/cmd/go-minitrace/cmds/query/commands_test.go — Added app-config-backed repository discovery tests for the `query commands` command tree

