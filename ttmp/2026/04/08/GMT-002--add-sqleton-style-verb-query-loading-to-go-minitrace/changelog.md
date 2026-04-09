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


## 2026-04-08

Implemented MinitraceCommand compilation from parsed specs, including path/source metadata propagation and optional bool flag normalization with regression tests (commit 00830a7).

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/compiler.go — Compiler added
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/compiler_test.go — Compiler regression coverage added


## 2026-04-08

Implemented repository-backed MinitraceCommand catalog loading with fstest-backed coverage for SQL commands, aliases, duplicate path precedence, and alias-target validation (commit 16fc1a6).

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/catalog.go — Catalog loader added
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/catalog_test.go — Catalog tests added


## 2026-04-08

Added an embedded built-in MinitraceCommand repository with initial read-only commands, an example alias, and a helper for loading the embedded catalog (commit 7cc5370).

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/assets.go — Embedded source-root helper added
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/core/framework-summary.sql — Built-in command asset added


## 2026-04-08

Implemented MinitraceCommand SQL rendering helpers and rendering tests, then wired an additive query commands CLI subgroup that loads the embedded catalog, reuses shared read-only validation, resolves aliases, renders SQL, and executes via DuckDB (commits afeb0a4, b218017).

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/cmd/go-minitrace/cmds/query/command_runtime.go — CLI runtime added
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/pkg/minitracecmd/render.go — Renderer added


## 2026-04-09

Expanded the remaining ticket work into detailed API/frontend subtasks, added a protobuf transport for query commands, implemented backend v2 query-command list/execute handlers with tests, and wired the frontend command sidebar/form/execution flow end to end (commits 6b78de0, b47f81c, 122c0dc).

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/cmd/go-minitrace/cmds/serve/handlers_query_commands_v2.go — Backend query-command handlers added
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/proto/go_go_golems/minitrace/api/v1/query_commands.proto — Query-command transport schema added
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/web/src/pages/QueryEditorPage.tsx — Frontend command selection and execution flow added


## 2026-04-09

Completed the ticket implementation loop by checking off all remaining detailed tasks, re-running docmgr validation, and uploading a refreshed document bundle to reMarkable under a new progress filename.

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/reference/01-investigation-diary.md — Diary updated with backend/frontend implementation and final validation steps
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/tasks.md — All detailed implementation tasks checked off


## 2026-04-09

Ticket closed


## 2026-04-09

Ran a manual end-to-end smoke test against a local serve instance to verify the new query-command API and browser flow, then closed the ticket after confirming command execution and alias-default behavior in the UI.

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/cmd/go-minitrace/cmds/serve/handlers_query_commands_v2.go — Manual API smoke test covered list and execute endpoints
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/ttmp/2026/04/08/GMT-002--add-sqleton-style-verb-query-loading-to-go-minitrace/reference/01-investigation-diary.md — Diary updated with the manual smoke-test and closure step
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/web/src/pages/QueryEditorPage.tsx — Manual browser smoke test covered command selection


## 2026-04-09

Added a follow-up SQL-debugging refinement for structured query commands: the API now exposes raw sqleton template SQL metadata, the web UI shows raw-template and last-rendered-SQL accordions, and the flow was revalidated with backend tests, web build, and a manual browser smoke test (commit 4076a50).

### Related Files

- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/proto/go_go_golems/minitrace/api/v1/query_commands.proto — Query-command transport refined with raw SQL metadata
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/web/src/components/QueryEditor/QueryCommandForm.tsx — UI debug-helper accordions added for raw and rendered SQL
- /home/manuel/workspaces/2026-04-08/sqleton-minitrace/go-minitrace/web/src/pages/QueryEditorPage.tsx — Page now stores last rendered command SQL for debug display

