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

