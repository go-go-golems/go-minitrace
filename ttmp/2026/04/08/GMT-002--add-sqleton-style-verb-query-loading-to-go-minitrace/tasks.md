# Tasks

## TODO

- [x] Create docmgr ticket workspace for sqleton-style verb query loading research
- [x] Create primary design doc and investigation diary documents
- [x] Investigate sqleton query-command docs, repository discovery, loader, spec, and smoke tests
- [x] Investigate go-minitrace query engine, serve query-library handlers, protobuf metadata contract, and frontend query editor
- [x] Write the detailed analysis / design / implementation guide for a new intern
- [x] Write the chronological investigation diary
- [x] Write the follow-up `MinitraceCommand` implementation-plan guide with glazed parameter-definition reuse
- [x] Update ticket index, tasks, and changelog
- [x] Relate key source files to the design doc and diary with `docmgr doc relate`
- [x] Run `docmgr doctor --ticket GMT-002 --stale-after 30`
- [x] Upload the ticket bundle to reMarkable with dry-run first
- [x] Verify the uploaded bundle in the remote folder listing
- [x] Implement pkg/minitracecmd core types, errors, and source-kind detection
- [x] Implement sqleton-style SQL preamble parsing with focused parser tests
- [x] Implement alias YAML parsing with focused parser tests
- [x] Implement MinitraceCommand compilation and bool-flag normalization with tests
- [x] Implement repository-backed catalog loading, precedence rules, and alias validation with tests
- [x] Add built-in MinitraceCommand repository assets for initial read-only commands
- [x] Implement SQL rendering helpers and MinitraceCommand rendering with tests
- [x] Add CLI query commands subgroup and runtime execution adapter
- [x] Add query-command API transport and serve handlers
- [x] Add frontend query-command types, sidebar integration, and form execution flow
- [x] Refresh diary, changelog, validation, and reMarkable bundle after implementation
- [x] Define query-command protobuf schema and add code generation for backend/frontend consumers
- [x] Implement backend catalog-to-API DTO conversion helpers for MinitraceCommand metadata and parameters
- [x] Add GET /api/v2/query-commands handler that lists embedded query commands and aliases
- [x] Add POST /api/v2/query-commands/{path...}/execute handler for alias resolution, rendering, readonly validation, and DuckDB execution
- [x] Add backend tests for query-command listing, execution, alias behavior, and error handling
- [x] Add frontend query-command TypeScript types and protobuf/API adapter helpers
- [x] Add frontend API client methods for listing and executing query commands
- [x] Add Commands section to the query sidebar and page state for selecting structured commands
- [x] Implement QueryCommandForm rendering for initial Glazed-backed field types and defaults
- [x] Wire query-command execution results into QueryEditorPage without regressing raw SQL flows
- [x] Add frontend stories/tests for query-command sidebar and form behavior
