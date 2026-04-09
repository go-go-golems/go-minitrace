# Changelog

## 2026-04-08

- Initial workspace created
- Investigated sqleton query-command docs, repository discovery, source-kind detection, neutral SQL command spec parsing, and smoke tests
- Investigated go-minitrace query engine, serve query-library handlers, protobuf query metadata contract, and frontend query editor architecture
- Wrote the primary design / analysis / implementation guide for sqleton-style repository-backed query verbs and UI query forms in go-minitrace
- Wrote the chronological investigation diary for the ticket

## 2026-04-08

Completed the research and writing pass: investigated sqleton repository-backed SQL command loading and compared it against go-minitrace's current raw SQL preset/query library model, then wrote the primary design guide and diary.

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/cmd/go-minitrace/cmds/serve/handlers_queries.go — Primary go-minitrace raw query-library reference
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/design-doc/01-sqleton-style-verb-query-loading-for-go-minitrace-analysis-design-and-implementation-guide.md — Main deliverable authored for this ticket
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/sqleton/pkg/cmds/spec.go — Primary sqleton spec/parser reference


## 2026-04-08

Validated the ticket with docmgr doctor and uploaded the document bundle to reMarkable under /ai/2026/04/08/GMT-002 after a successful dry-run.

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/design-doc/01-sqleton-style-verb-query-loading-for-go-minitrace-analysis-design-and-implementation-guide.md — Primary uploaded document
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/reference/01-investigation-diary.md — Primary uploaded supporting document
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/tasks.md — Delivery checklist updated to complete


## 2026-04-08

Added a second design doc that turns the chosen architecture into a concrete MinitraceCommand implementation plan with direct glazed parameter-definition reuse, then uploaded an updated reMarkable bundle including the new guide.

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/design-doc/02-minitracecommand-implementation-plan-with-glazed-parameter-definition-reuse.md — New implementation-plan deliverable
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/reference/01-investigation-diary.md — Diary updated with the MinitraceCommand follow-up step


## 2026-04-08

Expanded the MinitraceCommand implementation guide with a literal first-PR coding checklist, starter Go stubs, exact initial test names, acceptance criteria, and review instructions; then re-uploaded the updated bundle to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/design-doc/02-minitracecommand-implementation-plan-with-glazed-parameter-definition-reuse.md — Expanded with first-PR checklist and starter stubs
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/reference/01-investigation-diary.md — Diary updated with the checklist refinement step


## 2026-04-08

Refined the MinitraceCommand guide with literal first-PR starter stubs and uploaded a refreshed bundle to reMarkable under a new filename to avoid overwriting the earlier PDF.

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/design-doc/02-minitracecommand-implementation-plan-with-glazed-parameter-definition-reuse.md — Guide expanded with exact first-PR checklist and starter stubs


## 2026-04-08

Started implementation by adding the pkg/minitracecmd package core: sentinel errors, the MinitraceCommand/MinitraceCommandSpec types, validation rules, source-kind detection, and initial tests (commit b8f3229).

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/source_kind.go — Source-kind detection added for repository scanning
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/types.go — Core MinitraceCommand model added


## 2026-04-08

Implemented sqleton-style SQL command parsing for MinitraceCommand specs, including preamble splitting, YAML decoding into glazed-backed fields, and focused parser tests (commit 5acc6c5).

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/parse_sql.go — SQL parser and detection helpers added
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/parse_sql_test.go — Parser regression coverage added


## 2026-04-08

Implemented alias YAML parsing for MinitraceCommand specs, including direct capture of alias defaults and focused validation tests (commit 50f4d10).

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/parse_alias.go — Alias parser added
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/parse_alias_test.go — Alias parser tests added

