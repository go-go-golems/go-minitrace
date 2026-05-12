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

