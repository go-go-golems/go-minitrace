# Diary

## Goal

Implement built-in DuckDB querying in `go-minitrace` as a proper Glazed command group instead of relying only on external SQL files and manual `duckdb` CLI usage.

## Planned Steps

- inspect the current CLI/query surface and confirm the DuckDB Go integration path
- add detailed implementation tasks in the ticket
- scaffold a new `query` command group and `query duckdb` command using Glazed fields and sections
- add an internal query engine package and embedded preset SQL assets
- add help docs and tests
- validate against a real converted archive
- commit in reviewable checkpoints

## Step 1: Current-State Inspection

Confirmed:

- `go-minitrace` root command already follows the correct Glazed logging/help pattern
- there is no `query` command group yet
- query support currently exists only as SQL files under `queries/`
- the recommended Go driver is now the official DuckDB package at `github.com/duckdb/duckdb-go/v2`

Reference points used:

- `cmd/go-minitrace/main.go`
- `cmd/go-minitrace/cmds/common/build.go`
- `queries/*.sql`
- `queries/README.md`
- official DuckDB Go driver repository documentation

## Step 2: Task Planning

Added concrete GOQUERY implementation tasks in the ticket so the work can be completed in bounded slices:

- command scaffolding
- embedded preset assets
- loader/query engine
- help docs
- tests
- real smoke validation

Next step is the command scaffolding patch.

## Step 3: Query Command Scaffolding

Added the first code slice for the built-in query feature:

- `cmd/go-minitrace/cmds/query/root.go`
- `cmd/go-minitrace/cmds/query/duckdb.go`
- `pkg/doc/query.md`
- registration of the `query` group in `cmd/go-minitrace/main.go`

Important design choices in this slice:

- The new verb is a proper Glazed command, not a direct Cobra flag handler.
- Settings are modeled explicitly in `DuckDBQuerySettings`.
- Flags are declared with `fields.New(...)`.
- The command includes both the Glazed output section and the command-settings section.
- `RunIntoGlazeProcessor` currently validates mutually exclusive modes and emits a scaffold row so the command is already testable before the DuckDB engine lands.

Verification performed:

- `go test ./...`
- `go run ./cmd/go-minitrace query duckdb --archive-glob './output/active/*/*.minitrace.json' --load-only --output json`

The smoke command returned a structured scaffold row with the decoded settings, which confirms the new group is registered and the Glazed decoding path works.

Next step is the real DuckDB engine package plus embedded preset assets.
