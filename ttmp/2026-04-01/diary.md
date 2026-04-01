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

## Step 4: Embedded Presets And DuckDB Engine

Added the actual query backend:

- official DuckDB driver dependency via `github.com/duckdb/duckdb-go/v2`
- `pkg/query/assets.go`
- `pkg/query/engine.go`
- embedded preset SQL files under `pkg/query/presets/`
- command integration in `cmd/go-minitrace/cmds/query/duckdb.go`

Important implementation choices:

- The command opens DuckDB through Go instead of shelling out to the `duckdb` CLI.
- Query loading and execution use a single explicit `*sql.Conn`, which avoids temp-table visibility issues across pooled connections.
- The archive loader SQL is generated dynamically from Glazed settings instead of depending on the repo-local `queries/load.sql` file.
- Built-in presets are embedded into the binary and use `{{TABLE_NAME}}` substitution rather than hard-coded `sessions_base`.
- Row scanning normalizes `[]byte` and `time.Time` values before emitting Glazed rows.

This keeps the runtime fully inside the Go CLI and avoids hidden path or environment assumptions.

## Step 5: Tests And Real Smoke Validation

Added query tests in `pkg/query/engine_test.go`:

- preset resolution test
- SQL-file resolution test
- value normalization test
- end-to-end load-and-query integration test using a fixture minitrace archive written by the shared archive writer

Full validation performed:

- `go mod tidy`
- `go test ./...`
- `go build ./...`
- `go run ./cmd/go-minitrace query duckdb --archive-glob '/tmp/go-minitrace-chatgpt-json/active/*/*.minitrace.json' --preset session-list --output json`
- `go run ./cmd/go-minitrace query duckdb --archive-glob '/tmp/go-minitrace-chatgpt-json/active/*/*.minitrace.json' --preset framework-summary --output json`
- `go run ./cmd/go-minitrace query duckdb --archive-glob '/tmp/go-minitrace-chatgpt-json/active/*/*.minitrace.json' --sql 'SELECT COUNT(*) AS sessions FROM sessions_base' --output json`
- `go run ./cmd/go-minitrace query duckdb --archive-glob '/tmp/go-minitrace-chatgpt-json/active/*/*.minitrace.json' --sql-file <tmpfile> --output json`

Observed results:

- `session-list` returned the six expected ChatGPT JSON transcript sessions.
- `framework-summary` returned one `chatgpt-web` aggregate row with the expected counts and averages.
- both inline SQL and `--sql-file` returned `sessions = 6`.

## Step 6: Engine Checkpoint Commit

Checkpointed the backend slice as:

- `2d5ce7f` `Add DuckDB query engine and preset support`

That commit contains:

- the real `query duckdb` runtime path in `cmd/go-minitrace/cmds/query/duckdb.go`
- DuckDB open/load/query helpers in `pkg/query/`
- embedded preset SQL assets
- query engine tests

## Step 7: Documentation Sync

Synced the repository-facing documentation to the new command:

- `README.md`
- `pkg/doc/overview.md`
- `pkg/doc/query-duckdb.md`
- `ttmp/2026-04-01/tasks.md`
- `ttmp/2026-04-01/diary.md`

This keeps the user-visible command surface, the embedded help pages, and the implementation diary aligned with the shipped backend.
