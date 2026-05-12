# Changelog

## 2026-05-12

- Initial workspace created


## 2026-05-12

Created GMT-008 implementation ticket, ported the GMINI-0002 root-cause design, seeded tasks, and started the implementation diary.

### Related Files

- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert.go — Main converter implementation target identified by the design.
- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert_test.go — Regression-test target identified by the task list.


## 2026-05-12

Implemented turnsdb converter fixes and regression tests for metadata-stable tool-call deltas, tool-call dedupe/merge, ToolCallsInTurn linking, and blank text payload normalization (commit ce2d48f). Targeted go test ./pkg/adapters/turnsdb passes; full pre-commit go test ./... still has unrelated config-discovery failures in query/minitracecmd tests.

### Related Files

- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert.go — Implemented stable tool block fingerprints
- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert_test.go — Added regression coverage for metadata churn


## 2026-05-12

Smoke-checked real Coinvault conversion for session 8730... using local turns.sqlite: tool calls dropped from 12 duplicated entries to 6 unique successful entries, linked count is 6, pending no-result count is 0, and JSON-looking blank text artifacts are 0. Interleaving remains a model limitation: all six tool calls link to one assistant turn.

### Related Files

- /home/manuel/code/gec/2026-03-16--gec-rag/k3s-recovery/clean/coinvault-turns.sqlite — Local source fixture used for GMT-008 smoke validation.
- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert.go — Smoke-checked converter behavior against a real Coinvault turns.db session.


## 2026-05-12

Added Geppetto/Pinocchio turns identity research guide and follow-up tasks for simplifying go-minitrace turnsdb delta handling with semantic block identities and ordered conversion.

### Related Files

- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert.go — Follow-up implementation target for semantic block identity and ordered delta processing.
- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/ttmp/2026/05/12/GMT-008--fix-turnsdb-tool-call-conversion-and-coinvault-transcript-rendering/analysis/01-geppetto-turns-identity-and-minitrace-delta-handling-guide.md — New research guide.


## 2026-05-12

Uploaded the Geppetto turns identity guide to reMarkable as GMT-008_Geppetto_Turns_Identity_Guide.pdf under /ai/2026/05/12/GMT-008.

### Related Files

- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/ttmp/2026/05/12/GMT-008--fix-turnsdb-tool-call-conversion-and-coinvault-transcript-rendering/analysis/01-geppetto-turns-identity-and-minitrace-delta-handling-guide.md — Uploaded guide source.


## 2026-05-12

Simplified turnsdb semantic identity: non-tool blocks now require block_id, tool blocks require payload id, and LCS delta fails fast instead of falling back through legacy content/metadata heuristics. Targeted turnsdb tests and representative Coinvault conversion pass.

### Related Files

- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert.go — Fail-fast semantic block identity and LCS error propagation.
- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert_test.go — Updated tests to require IDs and added missing-identity failures.


## 2026-05-12

Committed strict turnsdb identity requirement: non-tool blocks must have block_id and tool blocks must have payload id; no content/metadata fallback chain (commit a0e2a9c).

### Related Files

- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert.go — Strict semantic identity implementation.
- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert_test.go — Missing-identity regression coverage.


## 2026-05-12

Refactored turnsdb conversion to process delta blocks in source order, added ordered text/tool fixtures, kept strict IDs, and audited the full local Coinvault turns.db. Full conversion succeeds for 31 sessions with 143 tool calls, 140 turn links, 0 pending no-result calls; the remaining 3 unlinked calls have no reliable assistant segment context and are documented as a model/data-shape caveat (commit f8bc060).

### Related Files

- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert.go — One-pass ordered delta processing and tool attachment logic.
- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert_test.go — Ordered interleaving


## 2026-05-12

Updated the transcript UI generic tool-call detail renderer to show tool call ID and structured arguments before output, fixing missing SQL tool arguments in the expanded row (commit 45bb2c0). Rebuilt embedded frontend and restarted the 8092 validation server; skipped Playwright because another user is using it.

### Related Files

- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/web/src/components/TranscriptViewer/ToolCallRow.tsx — Shows generic tool call identity and arguments in expanded details.


## 2026-05-12

Step 8: folded long system prompts by default in the transcript UI and validated the rebuilt frontend bundle (commit e5b733c).

### Related Files

- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/web/src/components/TranscriptViewer/BlockBody.tsx — Added reusable folded text rendering and default-collapsed system prompts.


## 2026-05-12

Step 9: diagnosed duplicate system prompt at turn #10 as regenerated system block IDs in turns.db and deduped system prompts by strict content_hash identity (commit c48124a).

### Related Files

- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert.go — System blocks now require content_hash for delta identity.
- /home/manuel/workspaces/2026-05-12/proper-minitrace-export-coinvault/go-minitrace/pkg/adapters/turnsdb/convert_test.go — Regression tests for regenerated system block IDs.

