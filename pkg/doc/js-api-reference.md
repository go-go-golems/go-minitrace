---
Title: JS API Reference
Slug: js-api-reference
Short: JavaScript runtime API available to minitrace JS command handlers
Topics:
- minitrace
- javascript
- js-commands
- js-api
- sqlite
- goja
Commands:
- query commands
Flags:
- archive-glob
- db-path
- table-name
- persist-loaded
IsTemplate: false
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

JavaScript command handlers run in a Goja-powered runtime with access to a `minitrace` module, Goja NodeJS-style helper modules, and scanner markers that turn `.js` files into typed `go-minitrace query commands` verbs.

The current API is **builder-composed**. Use fluent Go-backed builders to construct typed sources, policies, cache settings, query recipes, views, and session handles. The builders keep construction and validation on the Go side while view/query outputs remain plain JSON-serializable rows.

```js
const mt = require("minitrace");
```

> In xgoja apps this provider can be aliased as `require("mt")`. Query-command examples in this repository still use `require("minitrace")` because that is the provider's registered module name.

## Evidence-first JavaScript

Use JavaScript to compose sources and recipes, SQL to select normalized rows, and `mt.native()` when the question requires original JSONL request/compaction evidence. Native inspection is opt-in and does not populate normalized tables or execute historical commands.

```js
const mt = require("minitrace");
const db = mt.db().RuntimeArchives().SQLiteMemory()
  .MaxCellBytes(16000).StrictEvidence(true).Build();
try {
  const result = db.execute(mt.query().ToolRows().Build());
  if (result.error || !result.complete) {
    throw new Error(result.error || "Incomplete evidence");
  }
  return result.rows;
} finally {
  db.close();
}
```

`execute(recipe)` returns the same envelope as `queryResult(sql, ...args)`; it accepts a Go-backed recipe or `{sql: string, args: [...]}`. Query failures may be reported in `result.error`, while builder/argument failures can throw. Always inspect the envelope for analysis; `query()` is a rows-only convenience and cannot convey completeness diagnostics.

### Query fidelity and byte limits

`MaxCellBytes(n)` (or `mt.limits().CellBytes(n)`) limits UTF-8 bytes per result cell. `MaxCellChars` / `CellChars` remain deprecated byte-limit aliases, not Unicode character limits. A positive explicit byte limit takes precedence in query options. Clipping preserves valid UTF-8.

The result contains `columns`, `rows`, `count`, and `complete`, with optional `error`, `truncated`, `rowsTruncated`, and `cellDiagnostics`. `truncated` covers row overflow **or** lossy cell conversion. Each cell diagnostic gives zero-based `rowIndex` and `columnIndex`, `column`, `originalBytes`, `emittedBytes`, `truncated`, and optionally `invalidUtf8`.

With a three-byte cell limit, `SELECT 'abé' AS text` emits `ab`, reports four original bytes and two emitted bytes, and marks the result incomplete. `.StrictEvidence(true)` instead returns an error envelope with empty `rows` and `count: 0`; check `error`, do not interpret an empty result as zero matching data. SQL CLI users can select `--max-cell-bytes` and `--strict-evidence` too.

For read-only analysis without disk-cache writes, choose `.SQLiteMemory()` or the SQL CLI's `--cache-mode memory` / `none`. SQL CLI `auto` and `disk` modes can write a cache; `--cache-dir` makes its location explicit. Read-only queries do not imply a write-free process.

### Exact UTF-8 size

```js
mt.utf8ByteLength("abé"); // 4
```

This Go-backed helper computes UTF-8 byte length without constructing a temporary Buffer. It does not count tokens or replace a tokenizer. Node-style modules are partial compatibility surfaces: this addition does **not** implement `Buffer.byteLength` or promise full Node Buffer compatibility.

## Quick patterns

### Query-command default

```js
function sessionList(filters) {
  const mt = require("minitrace");
  const db = mt.db()
    .RuntimeArchives()
    .QueryCommandDefaults()
    .Build();
  try {
    return db.query(`
      SELECT session_id, title, agent_framework, turn_count, tool_call_count
      FROM sessions
      WHERE 1=1
      ${filters.framework?.length ? `AND agent_framework IN (${mt.sql.stringIn(filters.framework)})` : ""}
      ORDER BY started_at DESC
      LIMIT ${filters.limit ?? 25}
    `);
  } finally {
    db.close();
  }
}
```

### Explicit staged construction

```js
const sources = mt.sources()
  .RuntimeArchives()
  .Build();

const cache = mt.cache()
  .Memory()
  .Build();

const limits = mt.limits()
  .Rows(500)
  .CellChars(8000)
  .Build();

const db = mt.db()
  .Sources(sources)
  .Cache(cache)
  .Limits(limits)
  .Build();
```

### Session-oriented app flow

```js
const session = mt.session()
  .File(sessionPath)
  .InteractiveCache(cacheDir)
  .Open();
try {
  const transcript = session.view().Transcript().IncludeTools().Run();
  const timeline = session.view().Timeline().Run();
  const usage = session.view().TokenUsage().ByTurn().Run();
  return { summary: session.summary(), transcript, timeline, usage };
} finally {
  session.close();
}
```

### Upload/import flow

```js
const importer = mt.importer()
  .Content(uploadText)
  .Name(filename)
  .AutoDetect()
  .Strict()
  .Convert();

const preview = importer.Preview();
// Inspect preview.hasSystemPrompt, preview.sampleTurns, preview.sampleTools,
// preview.sampleEvents, preview.sampleAttachments, preview.eventCounts,
// preview.attachmentCounts, preview.subagentCount, and preview.hasImageSignals before saving.

const saved = importer
  .Into(sessionsDir)
  .SessionID(sessionId)
  .Save();
```

## Builder factories

| Factory | Purpose | Final value |
|---|---|---|
| `mt.importer()` | Detect, convert, and save uploaded content/files. | `SavedSession` or `ConvertedSession` maps. |
| `mt.sources()` | Build file/content/dir/glob/runtime source sets. | Go-owned `SourceSet`. |
| `mt.importPolicy()` | Build conversion strictness/auto-convert policy. | Go-owned `ImportPolicy`. |
| `mt.cache()` | Build cache policy. | Go-owned `CachePolicy`. |
| `mt.limits()` | Build query limits. | Go-owned `QueryLimits`. |
| `mt.db()` | Compose sources/policies/limits into a normalized SQLite DB handle. | `DBHandle`. |
| `mt.session()` | Open an opinionated transcript/session handle. | `SessionHandle`. |
| `mt.query()` | Build named SQL recipes. | `QueryRecipe`. |
| `mt.view()` | Build and run generic transcript/timeline/token view plans. | Plain rows/frames. |

## `mt.importer()`

Methods:

| Method | Description |
|---|---|
| `.Content(text)` | Use in-memory upload content. |
| `.File(path)` | Use a source file. |
| `.Name(name)` | Set original/source name. |
| `.SourcePath(path)` | Set source path metadata. |
| `.AutoDetect()` | Enable supported raw transcript conversion. |
| `.Format(format)` | Reserve a forced format hint for future use. |
| `.Strict(true|false)` | Set strict conversion behavior. |
| `.Into(rootDir)` | Set sessions root for `Save()`. |
| `.SessionID(id)` | Override saved session id. |
| `.Overwrite(true|false)` | Allow replacing an existing session directory. |
| `.Detect()` | Load/detect and return `{ format, adapter, diagnostics }`. |
| `.Convert()` | Convert and keep the Go-owned converted session in the builder. |
| `.Converted()` | Return a plain count-level summary of the converted session. |
| `.Preview()` | Return a transcript preview for validation: roles, tools, source events, attachments, sample turns/tool calls/events/attachments, system-prompt/thinking/image/subagent signals, and diagnostics. |
| `.Diagnostics()` | Return conversion diagnostics. |
| `.Save()` | Write `session.minitrace.json` and `metadata.json`. |

### Preview from the CLI

Use the CLI when you want the same importer preview behavior without writing JavaScript:

```bash
go-minitrace preview session --source-session /path/to/session.jsonl --output yaml
```

Directory/latest-N mode uses framework discovery and emits one row per previewed session:

```bash
go-minitrace preview session --framework codex --latest 5 --privacy structural --output json
go-minitrace preview session --framework claude-code --source-dir ~/.claude/projects --latest 3 --sample-limit 2 --output yaml
go-minitrace preview session --framework pi --source-dir ~/.pi/agent/sessions --latest 1
```

Preview privacy controls:

| Privacy | Behavior |
|---|---|
| `structural` | Counts, roles, tools, event/attachment kinds, booleans, and diagnostics only; content snippets/commands/paths are suppressed in samples. |
| `snippets` | Default; includes bounded snippets and commands for quick validation. |
| `full` | Includes full sampled turn text and command strings; use only on trusted local output paths. |

Important preview fields:

| Field | Meaning |
|---|---|
| `turnCount`, `toolCallCount` | Normalized conversation/action counts. |
| `eventCount`, `attachmentCount` | First-class source event/artifact counts. |
| `roleCounts`, `toolCounts` | Breakdown maps for turns and tools. |
| `eventCounts`, `attachmentCounts` | Breakdown maps by event kind and attachment kind. |
| `sampleTurns`, `sampleTools` | Bounded samples for validating role/tool conversion. |
| `sampleEvents`, `sampleAttachments` | Bounded samples for lifecycle facts and artifacts; raw JSON is not included. |
| `hasImageSignals` | True if turns, tools, events, or attachments include image-like signals. |

## `mt.sources()`

Methods: `.File(path)`, `.Archive(path)`, `.Files(paths)`, `.Dir(path)`, `.Glob(pattern)`, `.Content(text)`, `.Name(name)`, `.RuntimeArchives()`, `.Validate()`, `.Build()`.

`Build()` returns a Go-owned `SourceSet`. Pass it to `mt.db().Sources(...)` or `mt.session().Sources(...)`.

```js
const sources = mt.sources()
  .Dir("./output/active")
  .Glob("./extra/*.minitrace.json")
  .Build();
```

## Policy subbuilders

```js
const importPolicy = mt.importPolicy()
  .AutoConvert()
  .Strict()
  .Build();

const cache = mt.cache()
  .Auto()
  .Dir("./.cache/minitrace")
  .Build();

const limits = mt.limits()
  .Rows(1000)
  .Columns(128)
  .CellChars(6000)
  .TimeoutMs(5000)
  .Build();
```

### `mt.importPolicy()`

- `.AutoConvert(true|false)`
- `.NativeOnly()`
- `.Strict(true|false)`
- `.Lenient()`
- `.Format(format)`
- `.Build()`

### `mt.cache()`

- `.None()`
- `.Memory()`
- `.Disk()`
- `.Auto()`
- `.Dir(path)`
- `.ForceRebuild(true|false)`
- `.Build()`

### `mt.limits()`

- `.Rows(n)`
- `.Columns(n)`
- `.CellChars(n)`
- `.TimeoutMs(n)`
- `.RequireOrderBy(true|false)`
- `.Build()`

## `mt.db()`

`mt.db()` composes source sets and policies into a normalized SQLite DB handle.

Composition methods:

- `.Sources(sourceSet)`
- `.Import(importPolicy)`
- `.Cache(cachePolicy)` or legacy `.Cache("auto")`
- `.Limits(queryLimits)`
- `.Build()`

Convenience methods:

- source: `.File`, `.Archive`, `.Files`, `.Dir`, `.Glob`, `.Content`, `.RuntimeArchives`
- conversion: `.AutoConvert`, `.StrictConversion`, `.Strict`
- cache/storage: `.SQLiteMemory`, `.SQLiteDiskCache`, `.CacheAuto`, `.CacheDir`, `.ForceRebuild`
- limits: `.MaxRows`, `.MaxColumns`, `.MaxCellBytes`, `.StrictEvidence`, `.Timeout`, `.RequireOrderBy` (legacy byte alias: `.MaxCellChars`)
- presets: `.QueryCommandDefaults()`, `.InteractiveDefaults(cacheDir?)`
- introspection: `.Validate()`, `.CacheKey()`, `.sources()`

### DB handle

`Build()` returns a handle with:

| Method | Description |
|---|---|
| `query(sql, ...args)` | Execute a read-only SQLite query. |
| `queryOne(sql, ...args)` | Return the first row object, or `null` for no match; throw on query errors. |
| `execute(recipe)` | Execute a recipe with the full result/completeness envelope. |
| `queryResult(sql, ...args)` | Return result envelope including validation errors. |
| `schema()` / `tables()` / `stats()` | Schema introspection. |
| `sources()` | Sources used to build the handle. |
| `diagnostics()` | Conversion diagnostics. |
| `cacheInfo()` / `CacheInfo()` | Cache metadata. |
| `close()` | Close/release handle. |

Queries are validated as read-only SQL and bounded by row, column, cell, and timeout limits. Rows are objects keyed by column name or alias, not positional arrays. `query()` and `queryOne()` throw on query errors; `queryResult()` and `execute()` return an error envelope. Envelope `rows` and `columns` are always arrays, including failures. `execute()` accepts a built recipe or `{sql, args}`; use `queryResult(sql, ...args)` for a SQL string.

### Source events and attachments from JavaScript

Normalized SQLite now includes first-class source lifecycle facts and artifact references:

| Table | What it contains | How to use it from Goja |
|---|---|---|
| `events` | Derived renderable rows for turns/tools/annotations plus explicit source events. | Filter `kind NOT IN ('turn', 'tool_call', 'annotation')` when you want only source lifecycle facts. |
| `attachments` | Durable references to images/files/artifacts associated with sessions, turns, tools, or events. | Join/link by `tool_call_id`, `event_id`, `attachment_id`, or `turn_index`; do not expect blob bytes. |

Important source-event kinds include Pi `compaction`, `model_change`, `thinking_level_change`, Claude Code `mode_change`, `permission_mode_change`, `title_change`, `attachment`, and Codex `image_view`, `subagent_spawn`, `subagent_wait`, and `rate_limits`.

Concrete examples:

```js
// Codex viewed an image. Render this as an image marker or attachment card,
// not as a fake assistant text message.
{
  event_id: "event-image-1",
  kind: "image_view",
  title: "Codex image view",
  summary: "/tmp/screenshots/failure.png",
  tool_call_id: "call-view-image-1",
  attachment_id: "attachment-image-1"
}
{
  attachment_id: "attachment-image-1",
  kind: "image",
  name: "failure.png",
  media_type: "image/png",
  path: "/tmp/screenshots/failure.png",
  tool_call_id: "call-view-image-1",
  event_id: "event-image-1"
}

// Claude Code changed permission mode. Render this as a session/timeline badge
// so later edits can be interpreted with the right autonomy context.
{
  event_id: "event-permission-mode-1",
  kind: "permission_mode_change",
  title: "Permission mode changed",
  summary: "acceptEdits",
  severity: "info"
}

// Pi compacted the conversation. Render this as a context-boundary marker
// instead of scanning turn text for the word "compaction".
{
  event_id: "event-compaction-1",
  kind: "compaction",
  title: "Compaction",
  summary: "Conversation was compacted; prior context summarized.",
  collapsed_by_default: true
}

// Codex spawned a subagent and later waited for it. Render these as
// orchestration facts linked to their tool calls.
{
  event_id: "event-subagent-spawn-1",
  kind: "subagent_spawn",
  title: "Subagent spawned",
  summary: "agent_type=explorer",
  tool_call_id: "call-spawn-agent-1"
}
{
  event_id: "event-subagent-wait-1",
  kind: "subagent_wait",
  title: "Subagent completed",
  summary: "status=completed",
  tool_call_id: "call-wait-agent-1"
}
```

Query them from JavaScript like this:

```js
const session = mt.session()
  .File("/path/to/session.minitrace.json")
  .InteractiveCache(".cache/minitrace")
  .Open();
try {
  const sourceEvents = session.query(`
    SELECT event_id, timestamp, turn_index, kind, title, summary, tool_call_id, attachment_id
    FROM events
    WHERE kind NOT IN ('turn', 'tool_call', 'annotation')
    ORDER BY COALESCE(turn_index, 999999), COALESCE(ordinal, 999999), event_id
  `);
  const attachments = session.query(`
    SELECT attachment_id, timestamp, kind, name, media_type, path, url, tool_call_id, event_id, text_preview
    FROM attachments
    ORDER BY COALESCE(turn_index, 999999), timestamp, attachment_id
  `);
  return { sourceEvents, attachments };
} finally {
  session.close();
}
```

For UI code such as `ClubMedMeetup/minitrace-viz/lib/course-session-data.js`, the recommended pattern is to keep transcript messages derived from `turns` and `tool_calls`, then project source events/attachments into separate annotations, badges, or side panels. This avoids polluting the conversation text while still making compactions, subagents, mode changes, rate limits, and images visible.

## `mt.query()`

`mt.query()` builds SQL recipes. A recipe is a Go-owned object with `name()`, `sql()`, `args()`, `description()`, `output()`, and `toJSON()`.

Recipe selectors:

- `.SessionSummary()`
- `.SessionToolCounts()` — actual tool-row counts for every selected session, including zero-tool sessions
- `.TurnRows()`
- `.ToolRows()`
- `.FileRows()` — all materialized file-target evidence, including independent outcomes and provenance
- `.EventRows()`
- `.TurnBlockRows()`
- `.TokenUsageRows()`
- `.TranscriptRows()`
- `.TimelineRows()`

Modifiers:

- `.SessionID(id)`
- `.IncludeTools(true|false)`
- `.BySession()`
- `.ByTurn()`
- `.ByRole()`
- `.ByTool()`

Example:

```js
const recipe = mt.query()
  .TranscriptRows()
  .SessionID("sess-123")
  .IncludeTools()
  .Build();
const rows = db.query(recipe.sql(), ...recipe.args());
```

### Counts that retain zero-tool sessions

```js
const result = db.execute(mt.query().SessionToolCounts().Build());
if (result.error || !result.complete) throw new Error(result.error || "incomplete counts");
// Each row is {session_id, tool_count}, ordered by session_id.
// A session with no tool rows remains present with tool_count: 0.
const one = db.execute(mt.query().SessionToolCounts().SessionID("sess-123").Build());
```

The recipe starts from `sessions`, left-joins `tool_calls`, and uses `COUNT(tc.tool_call_id)`, not `COUNT(*)` or stored aggregate metadata. `.SessionID(id)` binds the filter; omit it for all loaded sessions. Counts describe normalized tool rows, including any orchestration records in that table. They do not assert process success or count only executions. Preserve the session identity set when comparing summaries: equal aggregate totals can hide a missing zero-tool session.

### Fidelity-aware projections

`ToolRows()` additionally exposes `source_ordinal`, `arguments_json`, and `output_status`. `source_ordinal` records the normalized archive tool-array position; it is not a native JSONL line or a result-arrival timestamp. Default recipe ordering remains emitting turn, timestamp, then tool ID. To recover archive-array order, explicitly order a projection by `source_ordinal` within a session. `arguments_json` is serialized input arguments; `output_status` is original archive output status, distinct from normalized `outcome_status`.

`TokenUsageRows()` adds `token_origin`, `token_unit`, `token_method`, and `reported_usage_fields`. Recorded usage rows use origin `recorded_usage` when any usage field is present, otherwise `unknown`; individual missing fields can still be zero-filled, so this is not proof of complete provider accounting. Tool output estimates use `estimated_tokens` and method `round_utf8_bytes_div_4`. Do not combine these estimates with recorded token totals as though they were the same measurement.


`SessionSummary()` includes the ordinary/orchestration/execution/file-change record counts, model invocation count, file-target count, and confirmed-target count. Total tool records are not the same as model invocations or executed processes.

`ToolRows()` preserves nullable `turn_index` and `success` and exposes `record_kind`, `outcome_status`, `framework_metadata_json`, `full_hash`, and `full_reference`. `file_targets_json` is the archive's structural input list; older scalar-only archives can have null here. Use `FileRows()` for the complete materialized target ledger, including explicitly labeled legacy scalar reports. File success is independent of tool success.

`TranscriptRows()` keeps unassociated tools at a null turn index and emits one row per tool even when source membership entries repeat. Rows are ordered by session and turn, with unassociated records last within each session.

`TimelineRows()` is a per-turn projection: it does not assign unassociated tools to a turn. Tool and file aggregates are computed separately so multiple targets or annotations cannot multiply tool counts. It reports `unknown_tool_count`, `file_touch_count`, and `confirmed_file_target_count`; `file_count` counts distinct paths, while `file_touch_count` counts evidence rows. Annotation presence is scoped to turn annotations.

Importer `Preview()` samples include `recordKind` and `fileTargets`. The shared Go helper `PreviewLoadedSessionWithOptions` also honors structural privacy: it retains target operations/evidence/outcomes but redacts path, native path, cwd and source reference without modifying the imported session. The JavaScript importer uses its existing default preview policy.

## `mt.view()` and `session.view()`

View builders run query recipes and return plain rows/frames.

Standalone view plans use `.DB(db)`:

```js
const frames = mt.view()
  .DB(db)
  .SessionID(sessionId)
  .TurnFrames()
  .IncludeThinking()
  .IncludeToolResults()
  .Run();
```

Session-bound view plans already know the DB and session id:

```js
const transcript = session.view()
  .Transcript()
  .IncludeTools()
  .Run();
```

Selectors:

- `.Transcript()`
- `.TurnFrames()`
- `.Timeline()`
- `.TokenUsage()`
- `.SessionSummary()`

Modifiers:

- `.IncludeTools(true|false)`
- `.IncludeThinking(true|false)`
- `.IncludeToolResults(true|false)`
- `.CollapseLongTextAt(chars)`
- `.BySession()` / `.ByTurn()` / `.ByRole()` / `.ByTool()`
- `.Plan()` to inspect the underlying query recipe
- `.Run()` to execute

### Turn frame identity and outcomes

`TurnFrames().Run()` returns frames keyed by both session and turn. Every frame includes `sessionId`, `turnIndex`, `unassociated`, `blocks`, `toolCalls`, and `stats`. Records with no proven turn are retained in a separate frame per session with `turnIndex: null` and `unassociated: true`, never assigned to turn zero. Associated frames sort by turn index, followed by the unassociated frame. Sparse or negative indexes do not require scanning every intervening integer.

`stats.failedToolCalls` counts only explicit false/zero outcomes. `stats.unknownToolCalls` counts absent binary outcomes, including pending/cancelled tools. Consumers must handle nullable frame indexes instead of treating every frame as a conversational turn.

```js
for (const frame of session.view().TurnFrames().Run()) {
  if (frame.unassociated) {
    // Display separately; do not invent an emitting message.
    renderUnassociated(frame.toolCalls);
  } else {
    renderTurn(frame.turnIndex, frame.blocks, frame.toolCalls);
  }
}
```

## `mt.session()`

`mt.session()` is the high-level transcript workflow. It applies transcript-friendly defaults and returns a lifecycle-bound session handle.

Builder methods:

- `.Sources(sourceSet)` / `.Source(sourceSet)`
- `.Import(importPolicy)`
- `.Cache(cachePolicy)`
- `.Limits(queryLimits)`
- `.SessionID(id)`
- `.File(path)`
- `.Content(text)`
- `.Name(name)`
- `.InteractiveCache(dir?)`
- `.Strict(true|false)`
- `.Open()`

Session handle methods:

- `id()`
- `summary()`
- `diagnostics()`
- `cacheInfo()`
- `db()`
- `query(sql, ...args)`
- `view()`
- `close()`

Always call `close()` in a `finally` block.

## SQL helpers

`mt.sql` contains string-literal helpers for building safe SQL fragments when query parameters are not ergonomic in generated SQL.

```js
mt.sql.string("O'Reilly")       // => "'O''Reilly'"
mt.sql.stringIn(["codex", "pi"]) // => "'codex', 'pi'"
mt.sql.like("tool fail")        // => "'%tool fail%'"
```

## Runtime metadata

`mt.runtime` includes query-command runtime settings. Only `archiveGlob` and `commandName` are meaningful; the rest are vestigial echoes of deprecated flags kept for backwards compatibility:

| Property | Type | Description |
|---|---|---|
| `archiveGlob` | `string[]` | Runtime archive globs consumed by `.RuntimeArchives()`. |
| `commandName` | `string` | Current command name. |
| `dbPath` | `string` | Vestigial: echoes the deprecated `--db-path` flag; has no effect on where queries run. |
| `tableName` | `string` | Vestigial: echoes the deprecated `--table-name` flag; do not use in new code. |
| `persistLoaded` | `boolean` | Vestigial: echoes the deprecated `--persist-loaded` flag. |

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Script expects `mt.query(sql)` to return rows | Script follows the removed ambient DuckDB API (`mt.query()` now builds named recipes). | Use `const db = mt.db().RuntimeArchives().QueryCommandDefaults().Build()` and call `db.query(sql)` against the normalized tables. |
| `query references disallowed table/view "sqlite_master"` | Script introspects SQLite system tables. | Use `db.schema()` or `db.tables()` instead. |
| Command fails and `--output json` shows `{"error": ...}` | JS errors are rendered as a compact one-line message plus a JSON envelope (`error`, `location`, `command`) on stdout. | Read `location` for the failing file:line; treat an `error` key as failure in automation. |
| `runtime archive glob is not configured` | `.RuntimeArchives()` used without runtime globs. | Pass `--archive-glob ...`, or use `.File()`, `.Dir()`, `.Glob()`, or `.Content()`. |
| Raw JSONL fails to load | Auto conversion disabled or strict conversion failed. | Use `.AutoConvert()` / `mt.importPolicy().AutoConvert().Build()` and inspect diagnostics. |
| View builder says DB is required | Standalone `mt.view()` was not bound to a DB. | Use `mt.view().DB(db)...` or `session.view()...`. |

## Related help

- `go-minitrace help structured-query-commands` — scanner markers and command metadata.
- `go-minitrace help query-commands` — query subsystem overview.
- `go-minitrace help query-duckdb` — migration notes for scripts written against the removed DuckDB engine.
- `testdata/query-repositories/js-showcase/` — worked JavaScript command examples.

## `mt.corpusManifest(paths)`

Preflight explicit normalized archive paths before selecting a corpus:

```js
const manifest = mt.corpusManifest(["/data/a.minitrace.json", "/data/b.minitrace.json"]);
if (manifest.conflictCount) throw new Error("Choose a source for each conflicting session");
const selected = manifest.entries
  .filter(entry => entry.duplicateOf === null)
  .map(entry => entry.path);
```

The result has `entries`, `sessions`, `conflictCount`, and `duplicateCount`. Entries include absolute `path`, normalized `sessionId`, nullable `originalSessionId` / `sourceFingerprint`, `sourceFormat`, raw archive `sha256`, `sizeBytes`, nullable zero-based `duplicateOf`, and `conflict`. Groups include `sessionId`, `entryIndexes`, `contentVariants`, and `conflict`.

Byte-identical copies are marked as duplicates. A normalized session ID with multiple byte hashes is a conflict, including differences only in whitespace or metadata. This conservative helper does not choose a winner, change ingestion, or silently substitute native identity for normalized identity. Supply selected paths to a source/DB builder explicitly. It reads each archive to compute identity; this is separate from the streaming native API.

## `mt.native()`

Inspect original JSONL separately from normalized archives:

```js
const scan = mt.native().File("/data/session.jsonl")
  .MaxRecordBytes(16 * 1024 * 1024)
  .ForEach(record => {
    if (record.diagnostic) return; // Record malformedness separately in real analysis.
    if (record.value.type === "api-request") {
      // Inspect record.source and record.value without evaluating contents.
    }
    // Return false for early termination; any other return continues.
  });
```

`ForEach` is synchronous and streaming. Every callback receives `source: {line, byteOffset, rawBytes}` and either `value` plus exact `rawJSON`, or `diagnostic: {code, message}`. Lines are one-based; offsets are zero-based raw byte positions and lengths include original line endings. Malformed JSON, invalid UTF-8 and oversized records are diagnosed. Oversized records are consumed/hash-counted without retaining their whole contents, then iteration continues.

The returned scan object contains `sha256`, `hashScope`, `bytes`, `lines`, `malformed`, and `stopReason`. Only EOF certifies `hashScope: "whole_source"`; an early stop has `"consumed_prefix"`. Hashes include exact consumed bytes even when records are malformed. Each operation closes its file on success, callback stop, callback error or cancellation. Cancellation is checked between reads, not by interrupting an OS read already blocked.

```js
const evidence = mt.native().File("/data/session.jsonl")
  .MaxRecordBytes(16 * 1024 * 1024)
  .MaxIndexBytes(64 * 1024 * 1024)
  .Inspect();
const incomplete = evidence.requests.filter(r => r.status !== "complete");
```

`Inspect()` builds an opt-in native index and returns `scan`, `requests`, `blobs`, `shapes`, `compactions`, `diagnostics`, `indexBytes`, and `limitations`. Defaults are 16 MiB per raw record and 64 MiB of logical retained index bytes. The index budget is not an exact Go heap ceiling; exceeding it throws instead of returning a partial success. Blob and shape maps retain source coordinates, decoded `value` and exact `rawJSON`. These may contain private transcript text; project appropriate fields before publication.

Requests expose `id`, `querySource`, `source`, `shapeHash`, `refs`, `status`, and `diagnostics`. A complete request requires an available shape, a valid nonnegative integer keep, an available retained base prefix, string tail references and resolvable blobs. Status is `complete` or the first diagnostic category (`missing_shape`, `missing_base`, `missing_blob`, `unsupported`); inspect the full diagnostics array for additional causes. Identical repeated blob/shape payloads deduplicate; conflicting identities retain the first object and make later referencing requests ambiguous.

Resolution is chronological and file-local. Later blobs do not retroactively repair earlier requests; cross-file inherited references are not recovered. Missing evidence is not zero context. No instruction-wrapper extractor is provided: application code must identify/version any provider-specific body parser it adds. Exact `rawJSON` preserves large numeric spelling; JavaScript's decoded `value` uses ordinary JS numeric semantics.

Compactions expose `boundary: {source, uuid, parentUuid?, type}`, optional `logicalParentUuid`, native compact metadata under optional `metadata`, `summaries`, and optional `beforeMessage`, `firstFollowingMessage`, and `firstFollowingAssistant`. The generated `NativeInspection` and related TypeScript interfaces describe these exact nested fields. Request `base` and `keep` are consumed during resolution but are not returned properties. Preserved-message metadata may refer to records whose physical order differs from conversation order. These are inspection coordinates, **not** certified causal replay cuts. Result-arrival provenance, model invocation, standalone-child identity conversion and agent-memory revival are outside this API.


### Native compaction projection

This example is executed by `TestNativeCompactionProjectionExample` from `pkg/minitracejs/testdata/native_compaction_projection.js` against a synthetic boundary, summary, attachment and EOF boundary. Optional fields remain absent in the native result; this projection intentionally converts absence to `null` for reporting.

```js
const inspection = mt.native().File("/data/session.jsonl").Inspect();
// `inspection` is the result of mt.native().File(path).Inspect().
// Preserve absence explicitly; physical coordinates do not certify replay.
inspection.compactions.map(c => ({
  boundary: c.boundary.source,
  boundaryUuid: c.boundary.uuid,
  logicalParentUuid: c.logicalParentUuid ?? null,
  metadata: c.metadata ?? null,
  summaries: c.summaries.map(s => ({uuid: s.uuid, source: s.source})),
  precedingUuid: c.beforeMessage?.uuid ?? null,
  firstFollowingUuid: c.firstFollowingMessage?.uuid ?? null,
  firstFollowingAssistantUuid: c.firstFollowingAssistant?.uuid ?? null,
}));
```

`metadata` retains provider-defined values, so inspect its structure before accessing nested fields in general-purpose code. `source` coordinates are physical JSONL locations. Neither a preserved-segment reference nor a first-following assistant establishes a replayable model checkpoint. Each `mt.native()` call creates a separate mutable builder; setters persist on that builder, and each `ForEach()`/`Inspect()` operation reopens its configured file.
