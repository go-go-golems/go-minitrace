# Tasks

## Phase 1 — Builder-Composed `go-minitrace` `mt` API

- [x] Add `mt.importer()` fluent builder with `Content`, `File`, `Name`, `AutoDetect`, `Format`, `Strict`, `Into`, `SessionID`, `Convert`, `Converted`, and `Save`.
- [x] Add `mt.sources()` fluent subbuilder that returns a Go-owned `SourceSet` from `File`, `Archive`, `Files`, `Dir`, `Glob`, `Content`, `Name`, and `RuntimeArchives`.
- [x] Add `mt.importPolicy()` fluent subbuilder that returns a Go-owned `ImportPolicy` from `AutoConvert`, `NativeOnly`, `Strict`, and `Lenient`.
- [x] Add `mt.cache()` fluent subbuilder that returns a Go-owned `CachePolicy` from `None`, `Memory`, `Disk`, `Auto`, `Dir`, and `ForceRebuild`.
- [x] Add `mt.limits()` fluent subbuilder that returns Go-owned `QueryLimits` from `Rows`, `Columns`, `CellChars`, `TimeoutMs`, and `RequireOrderBy`.
- [x] Refactor `mt.db()` into a composition builder that accepts `Sources(SourceSet)`, `Import(ImportPolicy)`, `Cache(CachePolicy)`, and `Limits(QueryLimits)`.
- [x] Keep concise `mt.db()` convenience methods such as `File`, `Content`, `Name`, `RuntimeArchives`, `AutoConvert`, `Strict`, `CacheAuto`, `QueryCommandDefaults`, and `InteractiveDefaults` where they delegate to internal subbuilders.
- [x] Add `mt.session()` fluent builder with `Sources`, `Source`, `Import`, `Cache`, `Limits`, `SessionID`, `File`, `Content`, `Name`, `InteractiveCache`, `Strict`, and `Open`.
- [x] Add lifecycle-safe session handles with `summary()`, `diagnostics()`, `cacheInfo()`, `db()`, `query()`, `view()`, and `close()`.

## Phase 2 — Query Recipe and View Plan Builders

- [x] Add `mt.query()` fluent builder with recipe selectors `SessionSummary`, `TurnRows`, `ToolRows`, `EventRows`, `TurnBlockRows`, `TokenUsageRows`, `TranscriptRows`, and `TimelineRows`.
- [x] Add query-builder modifiers `SessionID`, `IncludeTools`, `BySession`, `ByTurn`, `ByRole`, and `ByTool`.
- [x] Make `mt.query().<Recipe>().Build()` return a Go-owned `QueryRecipe` with `name()`, `sql()`, `args()`, `description()`, `output()`, and `toJSON()`.
- [x] Add `mt.view()` fluent builder with `DB`, `SessionID`, `Transcript`, `TurnFrames`, `Timeline`, `TokenUsage`, `SessionSummary`, and `Run`.
- [x] Add `session.view()` as a session-bound view builder.
- [x] Add view-builder modifiers `IncludeTools`, `IncludeThinking`, `IncludeToolResults`, `CollapseLongTextAt`, `BySession`, `ByTurn`, `ByRole`, and `ByTool`.
- [x] Keep view outputs plain JSON-serializable transcript rows, turn frames, token usage rows, and timeline rows.
- [x] Add output-contract tests for transcript rows, turn frames, token usage rows, timeline rows, and query recipe objects.

## Phase 3 — Documentation and Examples Cutover

- [x] Rewrite `go-minitrace/pkg/doc/js-api-reference.md` around builder factories: `mt.importer`, `mt.sources`, `mt.importPolicy`, `mt.cache`, `mt.limits`, `mt.db`, `mt.session`, `mt.query`, and `mt.view`.
- [x] Replace monolithic `mt.db().RuntimeArchives().Build()` examples with staged fluent examples and concise preset examples such as `mt.db().RuntimeArchives().QueryCommandDefaults().Build()`.
- [x] Add upload/import examples using `mt.importer().Content(...).Name(...).Into(...).SessionID(...).Save()`.
- [x] Add app-style examples using `mt.session().File(...).InteractiveCache(...).Open()` and `session.view().Transcript().IncludeTools().Run()`.
- [x] Update query-command tests and showcase repositories to the builder-composed API.
- [x] Search for stale `mt.db.open`, `mt.session.open`, `OpenDBOptions`, `SessionOpenOptions`, `mt.import.save`, `mt.queries.*`, `mt.views.*`, and old options-map examples.

## Phase 4 — `minitrace-viz` xgoja Wiring Cutover

- [x] Remove `clubmed-minitrace-viz` from `ClubMedMeetup/minitrace-viz/xgoja.yaml` packages.
- [x] Remove local `mt` module registration from `ClubMedMeetup/minitrace-viz/xgoja.yaml`.
- [x] Alias `go-minitrace` as `mt`.
- [x] Remove the separate `as: minitrace` module alias from the app.
- [x] Regenerate/rebuild xgoja output using the repository's standard build command.

## Phase 5 — `minitrace-viz` App Refactor

- [x] Replace `mt.source(...).detect().convert().save(...)` in `lib/session-service.js` with `mt.importer().Content(...).Name(...).Into(...).SessionID(...).Save()`.
- [x] Replace `mt.archiveFile(...).turnBlocks(...)` in `lib/timeline-data.js` with `mt.session().File(...).InteractiveCache(...).Open()` and `session.view().Timeline()` / `session.view().TurnFrames()`.
- [x] Update `lib/course-session-data.js` transcript construction to consume new transcript/timeline rows where appropriate.
- [x] Update `lib/course-session-data.js` context-window construction to consume new frame/token rows while keeping WidgetRenderer-specific teaching logic app-side.
- [x] Remove or rewrite `/analyze` using `mt.query()` recipes; do not keep the old report builder.
- [x] Remove or rewrite `/api/report/:sessionId` and `/api/report-presets`.
- [x] Update `/api/session/:sessionId/turn-blocks` to use `session.view().TurnFrames()` or remove it if no longer needed.
- [x] Smoke-test upload, sessions list, timeline, transcript-data, and context-window-data endpoints.

## Phase 6 — Delete Old ClubMed `mtapi`

- [x] Delete `ClubMedMeetup/minitrace-viz/pkg/mtapi/*.go`.
- [x] Delete `ClubMedMeetup/minitrace-viz/pkg/mtapiprovider/provider.go`.
- [x] Remove unused dependencies from `ClubMedMeetup/minitrace-viz/go.mod` and generated module wiring.
- [x] Search the workspace for `mt.source`, `mt.archiveFile`, `reportPresets`, `pkg/mtapi`, and `pkg/mtapiprovider`.
- [x] Keep only ticket/design docs as historical references to old APIs.

## Phase 7 — Validation and Handoff

- [x] Run `go test ./pkg/minitracedb ./pkg/minitracejs/... ./cmd/go-minitrace/cmds/query -count=1` in `go-minitrace`.
- [x] Run the minitrace-viz build/regeneration command.
- [x] Launch the minitrace-viz site with temporary sessions/cache directories.
- [x] Upload representative Pi fixture through the minitrace-viz smoke test; Codex, Claude Code, and native minitrace app uploads remain follow-up validation.
- [x] Validate transcript ordering, tool-call attachment, failed-tool markers, token totals, and context-window turn selection at smoke-test/API level.
- [x] Update changelog and diary with final implementation notes, commands, failures, and review instructions.
