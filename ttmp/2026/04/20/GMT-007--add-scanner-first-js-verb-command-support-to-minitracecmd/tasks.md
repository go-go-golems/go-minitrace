# Tasks

## Completed

- [x] Create the GMT-007 ticket workspace and core documents
- [x] Gather line-anchored evidence from the current go-minitrace and go-go-goja command/runtime code
- [x] Write the primary analysis/design/implementation guide for a new intern
- [x] Maintain the investigation diary and ticket bookkeeping
- [x] Validate the ticket docs and upload the bundle to reMarkable

## Remaining Implementation Work

### 1) Source-kind, command-model, and catalog changes for JS command files

- [x] Add `SourceJSCommand` support to `pkg/minitracecmd/source_kind.go`
- [x] Extend the command model in `pkg/minitracecmd/types.go` so verbs can be SQL-backed or JS-backed
- [x] Add explicit validation rules for JS-backed verbs in `pkg/minitracecmd/types.go`
- [x] Extend compiler output in `pkg/minitracecmd/compiler.go` to preserve JS execution metadata
- [x] Add JS parse/scan adapter code in `pkg/minitracecmd/parse_javascript.go`
- [x] Reuse `go-go-goja/pkg/jsverbs` scanning to extract `__package__`, `__section__`, and `__verb__` metadata
- [x] Convert scanned JS verbs into `MinitraceCommandSpec` values while preserving file/folder source information
- [x] Allow multiple verbs per JS file during scan/compile
- [x] Reject duplicate final command paths across SQL and JS sources with a clear catalog-load error
- [x] Extend `pkg/minitracecmd/catalog.go` to load JS command files alongside SQL and alias files
- [x] Add catalog-level tests for mixed SQL/JS repositories and duplicate-path failures

### 2) JS execution branch and host minitrace runtime API

- [ ] Refactor `cmd/go-minitrace/cmds/query/command_runtime.go` into a runtime-kind dispatcher
- [ ] Preserve the current SQL execution path unchanged for SQL-backed commands
- [ ] Add a JS execution branch that runs after archive loading and alias resolution
- [ ] Create or wire a Goja runtime builder for minitrace command execution
- [ ] Expose a minimal host API for JS commands (at least `query(...)`, likely `queryOne(...)`, plus SQL helper functions)
- [ ] Pass command/runtime/value context into JS handlers in a stable shape
- [ ] Reuse `pkg/jsverbs/runtime.go` patterns for module loading, handler lookup, and Promise handling
- [ ] Normalize JS results into Glazed rows for row-producing commands
- [ ] Decide and document whether text-mode JS commands are deferred or supported in the first implementation slice
- [ ] Add runtime tests for plain-object, array-of-object, primitive, and Promise-returning JS handlers
- [ ] Add failure-path tests for missing handler functions, thrown JS errors, and rejected Promises

### 3) Tests, docs, and smoke validation for mixed SQL/JS command repositories

- [x] Add parser/scanner tests for valid JS command files and invalid static metadata
- [ ] Add command-group/help tests showing JS commands appear under `query commands`
- [ ] Add alias tests proving YAML aliases can target JS-backed commands
- [ ] Add integration tests for a repository containing both SQL and JS command files
- [ ] Add end-to-end execution tests using a loaded archive fixture and a JS-backed command
- [ ] Update `pkg/doc/structured-query-commands.md` to document `.js` / `.cjs` scanner-first command files
- [ ] Update any command reference/help pages that describe supported repository source types
- [ ] Add at least one worked JS command example to repo docs or testdata
- [x] Run `go test ./...` in `go-minitrace`
- [ ] Run focused `go test ./...` coverage in `go-go-goja` for any touched JS runtime/scanner integration code
- [ ] Perform CLI smoke runs for mixed SQL/JS command repos and capture example commands in docs or diary
- [ ] Re-run `docmgr doctor --ticket GMT-007 --stale-after 30` after implementation docs are updated

