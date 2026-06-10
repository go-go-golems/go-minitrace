---
Title: Session Events and Attachments Design and Implementation Guide
Ticket: session-events-attachments
Status: active
Topics:
    - minitrace
    - architecture
    - transcript
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/adapters/claudecode/convert.go
      Note: Claude Code adapter source lifecycle and attachment records
    - Path: pkg/adapters/codex/convert.go
      Note: Codex adapter tool and lifecycle signals including view_image and subagents
    - Path: pkg/adapters/pi/convert.go
      Note: Pi adapter source lifecycle records to map to first-class events
    - Path: pkg/minitrace/builders.go
      Note: Session and child constructors that define default slice behavior
    - Path: pkg/minitrace/schema.go
      Note: Canonical session schema that will receive Events and Attachments
    - Path: pkg/minitracedb/convert.go
      Note: Auto-loader dispatch point for native minitrace JSON and source JSONL adapters
    - Path: pkg/minitracedb/materialize.go
      Note: Materialization flow that synthesizes derived events and will insert explicit events and attachments
    - Path: pkg/minitracedb/schema.go
      Note: Normalized SQLite schema including existing events table and future attachments table
ExternalSources: []
Summary: Design and implementation guide for first-class source events and attachments in minitrace sessions.
LastUpdated: 2026-06-10T19:50:00-04:00
WhatFor: Use this when implementing or reviewing first-class Event and Attachment support across minitrace schema, SQLite materialization, adapters, validation, and preview APIs.
WhenToUse: Use before changing session schema, mapping Pi/Codex/Claude lifecycle records, or exposing source artifacts in Goja/CLI previews.
---


# Session Events and Attachments Design and Implementation Guide

## Executive summary

Minitrace currently has strong primitives for conversational turns, tool calls, annotations, handovers, metrics, and session-level metadata. It also has a SQLite `events` table used by materialization to create a renderable timeline from turns, tool calls, and annotations. What it does **not** yet have is a canonical in-memory/native JSON representation for source-observed lifecycle events or durable attachments. As a result, adapters either drop source records, compress them into `framework_config`, or preserve them as annotations even when they are not really human/derived review notes.

This ticket adds two first-class session primitives:

- `Session.Events []Event`: source-observed lifecycle and timeline facts such as Pi compactions, Claude permission-mode changes, Claude title updates, Codex rate-limit observations, or explicit subagent lifecycle signals.
- `Session.Attachments []Attachment`: durable referenced artifacts such as images viewed by Codex, Claude Code attachment records, uploaded files, downloaded artifacts, or future code-interpreter outputs.

The implementation should remain backend-first and compatibility-safe. Existing `.minitrace.json` files must continue to decode because the new arrays are optional. Existing turn/tool/annotation-derived timeline rows should continue to appear in SQLite. Explicit source events should be inserted **in addition** to those derived rows, using stable event IDs so repeated materialization is deterministic.

## Problem statement and scope

Agent transcript formats contain more than messages and tool calls. Pi emits `session_info`, `custom`, and `compaction` records. Claude Code emits `mode`, `permission-mode`, `ai-title`, and `attachment` records. Codex emits token/rate-limit events, image-view tool calls, and subagent-oriented tool calls. Some of these facts are useful for later analysis, but they are not conversational turns and they are not necessarily operator annotations.

Today those records are handled unevenly:

- some are stored in `OperationalContext.FrameworkConfig`;
- some become annotations;
- some are only visible through raw JSON attached to a turn or tool call;
- some are discarded because there is no dedicated destination.

The scope of this ticket is to add a durable model and persistence path for these facts. It is **not** a full UI redesign. Frontend/protobuf rendering can follow later if needed. The backend schema, adapters, tests, docs, and preview surface should be sufficient for CLI and Goja users to load, inspect, and query the new data.

## Current-state architecture

### Canonical session schema

The canonical Go model lives in `pkg/minitrace/schema.go`. `Session` currently includes top-level metadata and these major arrays:

```go
// pkg/minitrace/schema.go:3-25, observed 2026-06-10
type Session struct {
    ID                 string
    SchemaVersion      string
    Profile            string
    // ... metadata omitted ...
    Turns              []Turn
    ToolCalls          []ToolCall
    Outcome            *Outcome
    Annotations        []Annotation
    Metrics            Metrics
}
```

Important existing structures:

- `Turn` captures conversation role, source, content, thinking, tool call IDs, streaming flags, and usage (`pkg/minitrace/schema.go:99-114`).
- `ToolCall` captures normalized tool operation, arguments, output, context, framework metadata, and spawned-agent metadata (`pkg/minitrace/schema.go:136-147`).
- `Annotation` captures human or derived review notes with scope, content, taxonomy, and classification (`pkg/minitrace/schema.go:190-198`).

There is no top-level `Events` array and no top-level `Attachments` array.

### Builder defaults

`pkg/minitrace/builders.go` constructs safe default sessions and child objects. `BuildSessionSkeleton` initializes `Turns`, `ToolCalls`, and `Annotations` as empty slices (`pkg/minitrace/builders.go:66-70`). Any new arrays should follow the same convention to avoid `null` surprises in generated JSON and tests.

### SQLite materialization

The normalized SQLite schema is defined in `pkg/minitracedb/schema.go`. `Tables()` registers the current tables in this order:

```text
sessions -> turns -> tool_calls -> turn_tool_calls -> files -> annotations -> handovers -> metrics -> events
```

The `events` table already exists (`pkg/minitracedb/schema.go:511-549`). It stores timeline rows with fields such as `event_id`, `turn_index`, `ordinal`, `kind`, `role`, `tool_call_id`, `annotation_id`, `title`, `summary`, `text`, `severity`, and `raw_json`.

Materialization lives in `pkg/minitracedb/materialize.go`. `MaterializeSession` currently inserts:

1. the session row;
2. each turn plus a derived turn event;
3. each tool call plus a derived tool-call event;
4. each annotation plus a derived annotation event;
5. handovers and metrics.

The relevant flow is:

```text
MaterializeSession
  insertSession
  for turn in session.Turns:
    insertTurn
    insertTurnEvent
  for toolCall in session.ToolCalls:
    insertToolCall
    insertToolCallEvent
  for annotation in session.Annotations:
    insertAnnotation
    insertAnnotationEvent
  insertHandover
  insertMetrics
```

This means the DB already has a timeline concept, but that timeline is synthesized from other objects. Adding `Session.Events` should reuse the table rather than inventing a parallel `source_events` table unless a future migration requires separate event namespaces.

### Auto-loading and adapters

`pkg/minitracedb/convert.go` is the central auto-loader. `LoadSessionContentAuto` first attempts native minitrace JSON, then parses JSONL and dispatches to Pi, Codex, or Claude Code adapters (`pkg/minitracedb/convert.go:54-92`). `DetectJSONLFormat` currently recognizes Pi from `session`, `model_change`, `thinking_level_change`, or `message`; Codex from `session_meta`, `turn_context`, `response_item`, or `event_msg`; and Claude Code from `system`, `user`, or `assistant` records with a message (`pkg/minitracedb/convert.go:94-112`).

Adapter observations:

- Pi adapter (`pkg/adapters/pi/convert.go`) handles `session`, `model_change`, `thinking_level_change`, and `message`. It stores model/thinking changes in `modelInfo` and framework config, but does not yet preserve `session_info`, `custom`, or `compaction` as timeline facts.
- Claude Code adapter (`pkg/adapters/claudecode/convert.go`) already preserves `mode`, `permission-mode`, and `ai-title` in framework config/title behavior. It currently maps `attachment` records to annotations (`pkg/adapters/claudecode/convert.go:136-139`, helper around `buildClaudeAttachmentAnnotation`).
- Codex adapter (`pkg/adapters/codex/convert.go`) has rich tool-call semantics and captures rate-limit metadata in `codexMetadata.LatestRateLimits`. It maps `view_image` as a tool-call/content-origin signal, but there is no artifact model for the viewed image.

### Validation and querying

The native JSON validator in `pkg/validate/json.go` focuses on annotation shape. Existing query docs mention JSON arrays for `turns`, `tool_calls`, and `annotations`; normalized SQL docs mention the existing `events` table. This ticket must update both validation and docs so users can rely on `events` and `attachments` in native JSON and SQLite.

## Gap analysis

### Gap 1: source lifecycle records have no canonical home

A Pi compaction is a real event in the session timeline. It indicates a context-management step and often lists modified files or edited data. Storing it as an annotation confuses source data with derived review commentary. Dropping it loses useful context.

### Gap 2: attachments are artifacts, not events and not annotations

A Claude Code attachment or Codex image view is best modeled as an artifact with identity and media metadata. It may also have a timeline event saying that the artifact was attached or viewed, but the artifact itself should remain queryable independently.

### Gap 3: the existing events table cannot currently distinguish explicit source events from derived timeline rows

The DB can store event rows, but every current row is derived from a turn/tool/annotation. If adapters start adding explicit events, rows need enough columns to preserve event metadata and optionally link to attachments.

### Gap 4: preview users need counts and samples without raw transcript dumps

The previous session-import work added privacy-safe previews. Events and attachments must follow the same principle: default output should show structure, counts, IDs, kinds, titles, and media types, not full raw payloads.

## Proposed architecture

### Conceptual model

```text
                           +-------------------+
                           | minitrace.Session |
                           +-------------------+
                              |     |      |
             conversation ----+     |      +---- review/derived notes
                              |     |                |
                          Turns     |           Annotations
                                    |
                  source facts -----+----- artifacts
                       |                         |
                    Events                 Attachments
```

Use these definitions consistently:

- **Turn**: a conversational message-like unit with role/content.
- **ToolCall**: an action requested or performed by the agent/tool layer.
- **Annotation**: a human or derived note about a session/turn/tool/event/attachment.
- **Event**: a source-observed or renderable timeline fact.
- **Attachment**: a durable artifact reference or payload descriptor.

### Proposed Go API

Add fields to `Session`:

```go
type Session struct {
    // existing fields...
    Turns       []Turn       `json:"turns"`
    ToolCalls   []ToolCall   `json:"tool_calls"`
    Events      []Event      `json:"events,omitempty"`
    Attachments []Attachment `json:"attachments,omitempty"`
    Outcome     *Outcome     `json:"outcome"`
    Annotations []Annotation `json:"annotations"`
    Metrics     Metrics      `json:"metrics"`
}
```

Add `Event`:

```go
type Event struct {
    ID                 string  `json:"id"`
    Timestamp          *string `json:"timestamp,omitempty"`
    TurnIndex          *int    `json:"turn_index,omitempty"`
    Ordinal            *int    `json:"ordinal,omitempty"`
    Kind               string  `json:"kind"`
    Role               string  `json:"role,omitempty"`
    ToolCallID         *string `json:"tool_call_id,omitempty"`
    AnnotationID       *string `json:"annotation_id,omitempty"`
    AttachmentID       *string `json:"attachment_id,omitempty"`
    Title              string  `json:"title,omitempty"`
    Summary            string  `json:"summary,omitempty"`
    Text               string  `json:"text,omitempty"`
    Severity           string  `json:"severity,omitempty"`
    CollapsedByDefault bool    `json:"collapsed_by_default,omitempty"`
    FrameworkMetadata  any     `json:"framework_metadata,omitempty"`
    RawJSON            any     `json:"raw_json,omitempty"`
}
```

Add `Attachment`:

```go
type Attachment struct {
    ID                string  `json:"id"`
    Timestamp         *string `json:"timestamp,omitempty"`
    Kind              string  `json:"kind"`
    Name              string  `json:"name,omitempty"`
    MediaType         string  `json:"media_type,omitempty"`
    Path              string  `json:"path,omitempty"`
    URL               string  `json:"url,omitempty"`
    SizeBytes         *int    `json:"size_bytes,omitempty"`
    Hash              string  `json:"hash,omitempty"`
    ContentRef        string  `json:"content_ref,omitempty"`
    TextPreview       string  `json:"text_preview,omitempty"`
    TurnIndex         *int    `json:"turn_index,omitempty"`
    ToolCallID        *string `json:"tool_call_id,omitempty"`
    EventID           *string `json:"event_id,omitempty"`
    FrameworkMetadata any     `json:"framework_metadata,omitempty"`
    RawJSON           any     `json:"raw_json,omitempty"`
}
```

Important constraints:

- Do not inline large blobs by default.
- Prefer `Path`, `URL`, `Hash`, `ContentRef`, and bounded `TextPreview`.
- Preserve raw source shape in `RawJSON` only when safe and bounded by normal JSON output expectations.
- Use `FrameworkMetadata` for normalized source-specific metadata that callers can inspect without reverse-engineering raw records.

### Builder helpers

Adapters should not manually assemble every field. Add helpers similar to `BuildAnnotation`:

```go
func BuildEvent(id string, timestamp *string, kind string, title string, summary string, raw any) Event
func BuildAttachment(id string, timestamp *string, kind string, name string, mediaType string, raw any) Attachment
```

Then adapters can set optional relationship fields:

```go
event := minitrace.BuildEvent("pi-compaction-000001", ts, "compaction", "Pi compaction", summary, record)
event.Role = "system"
event.CollapsedByDefault = true
event.FrameworkMetadata = map[string]any{"modified_file_count": len(files)}

attachment := minitrace.BuildAttachment("codex-image-call-123", ts, "image", path.Base(imagePath), mediaType, payload)
attachment.Path = minitrace.NormalizePath(imagePath)
attachment.ToolCallID = &toolCall.ID
```

### SQLite schema changes

Extend `events` with columns for explicit event metadata:

- `timestamp TEXT`
- `attachment_id TEXT`
- `framework_metadata_json TEXT`

Keep existing columns because they already serve derived turn/tool/annotation rows.

Add `attachments` table:

```sql
CREATE TABLE IF NOT EXISTS attachments (
    session_id TEXT NOT NULL,
    attachment_id TEXT NOT NULL,
    timestamp TEXT,
    kind TEXT,
    name TEXT,
    media_type TEXT,
    path TEXT,
    url TEXT,
    size_bytes INTEGER,
    hash TEXT,
    content_ref TEXT,
    text_preview TEXT,
    turn_index INTEGER,
    tool_call_id TEXT,
    event_id TEXT,
    framework_metadata_json TEXT,
    raw_json TEXT,
    PRIMARY KEY (session_id, attachment_id)
);
```

Recommended indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_attachments_kind ON attachments(kind);
CREATE INDEX IF NOT EXISTS idx_attachments_tool_call ON attachments(session_id, tool_call_id);
CREATE INDEX IF NOT EXISTS idx_attachments_event ON attachments(session_id, event_id);
```

### Materialization flow

After inserting adapters' primary objects, insert explicit events and attachments:

```text
MaterializeSession
  insertSession
  insert turns + derived turn events
  insert tool calls + derived tool-call events
  insert annotations + derived annotation events
  insert explicit session.Events
  insert session.Attachments
  insert handovers
  insert metrics
```

The explicit event insertion should use a helper, not direct SQL embedded in every adapter:

```go
func insertExplicitEvent(ctx context.Context, tx *sql.Tx, sessionID string, event minitrace.Event, ordinal int) error {
    eventID := event.ID
    if eventID == "" {
        eventID = fmt.Sprintf("event-%06d", ordinal)
    }
    dbOrdinal := nullableIntPointer(event.Ordinal)
    if dbOrdinal == nil {
        dbOrdinal = &ordinal
    }
    return insertRow(ctx, tx, "events", []string{
        "session_id", "event_id", "timestamp", "turn_index", "ordinal", "kind", "role",
        "tool_call_id", "annotation_id", "attachment_id", "title", "summary", "text",
        "severity", "collapsed_by_default", "framework_metadata_json", "raw_json",
    },
        sessionID, eventID, nullableString(event.Timestamp), nullableIntPointer(event.TurnIndex), dbOrdinal,
        event.Kind, event.Role, nullableString(event.ToolCallID), nullableString(event.AnnotationID),
        nullableString(event.AttachmentID), event.Title, event.Summary, event.Text,
        firstNonEmpty(event.Severity, "info"), boolInt(event.CollapsedByDefault),
        jsonNullable(event.FrameworkMetadata), mustJSON(event),
    )
}
```

Attachment insertion should be analogous.

### Adapter mapping plan

#### Pi

Pi should map non-message records as explicit source events:

| Source record | Existing behavior | New behavior |
|---|---|---|
| `session_info` | not handled | set `Session.Title`, add `Event{kind:"session_info"}` |
| `custom` | not handled | add `Event{kind:"custom.<customType>"}` with `FrameworkMetadata` payload summary |
| `compaction` | not handled | add `Event{kind:"compaction"}` with modified file count and privacy-safe summary |
| `model_change` | updates model config | keep config behavior; add `Event{kind:"model_change"}` |
| `thinking_level_change` | updates framework config | keep config behavior; add `Event{kind:"thinking_level_change"}` |

Pseudocode:

```go
for ordinal, record := range records {
    switch record["type"] {
    case "session_info":
        if name := stringValue(record["name"]); name != "" { explicitTitle = name }
        events = append(events, buildPiEvent(record, ordinal, "session_info", "Pi session info", name))
    case "custom":
        customType := firstNonEmpty(stringValue(record["customType"]), "unknown")
        events = append(events, buildPiEvent(record, ordinal, "custom."+customType, "Pi custom record", customType))
    case "compaction":
        details := mapValue(record["details"])
        summary := summarizePiCompaction(details)
        events = append(events, buildPiEvent(record, ordinal, "compaction", "Pi compaction", summary))
    }
}

session.Events = events
if explicitTitle != "" { session.Title = &explicitTitle }
```

#### Claude Code

Claude Code should keep existing config/title behavior while adding events and attachments:

| Source record | Existing behavior | New behavior |
|---|---|---|
| `mode` | `framework_config.mode` | also `Event{kind:"mode_change"}` |
| `permission-mode` | `framework_config.permission_mode`; autonomy mapping | also `Event{kind:"permission_mode_change"}` |
| `ai-title` | `Session.Title`; `framework_config.ai_title` | also `Event{kind:"title_change"}` |
| `attachment` | annotation | `Attachment` plus `Event{kind:"attachment"}`; optional compatibility annotation only if needed |

Pseudocode:

```go
case "attachment":
    attachment := buildClaudeAttachment(record, attachmentCounter)
    event := buildClaudeAttachmentEvent(record, attachment.ID, attachmentCounter)
    attachment.EventID = &event.ID
    attachments = append(attachments, attachment)
    events = append(events, event)
```

#### Codex

Codex should avoid changing the already-good tool-call model. Add artifacts/events around it:

| Source signal | New behavior |
|---|---|
| `view_image` function/tool call | add `Attachment{kind:"image", tool_call_id:<call>}` |
| `spawn_agent` function/tool call | add `Event{kind:"subagent_spawn", tool_call_id:<call>}` |
| `wait_agent` function/tool call | add `Event{kind:"subagent_wait", tool_call_id:<call>}` |
| `token_count` with `rate_limits` | add collapsed `Event{kind:"rate_limits"}` or keep config-only in Phase 1 |
| `task_started` / turn context | optional `Event{kind:"task_started"}` if useful for timeline rendering |

## Decision records

### Decision: add both events and attachments

- **Context:** Source formats contain both lifecycle facts and artifact references. Using one structure for both would force awkward fields or make artifact queries hard.
- **Options considered:** events only, attachments only, both structures, annotations only.
- **Decision:** Add both `Session.Events` and `Session.Attachments`.
- **Rationale:** Events answer “what happened when?” Attachments answer “what artifact existed or was referenced?” Both are needed for Claude/Codex/Pi fidelity.
- **Consequences:** More schema surface and tests, but clearer adapter mappings and less annotation abuse.
- **Status:** accepted for this ticket.

### Decision: reuse the existing SQLite `events` table

- **Context:** `pkg/minitracedb` already has an events table and derived event insert helpers.
- **Options considered:** create `source_events`, overload annotations, or reuse/extend `events`.
- **Decision:** Reuse and extend `events`.
- **Rationale:** Query users already have a timeline table. Explicit events can coexist with derived turn/tool/annotation events if IDs and `kind` values remain stable.
- **Consequences:** Need to prevent ID collisions with derived prefixes (`turn-`, `tool-`, `annotation-`). Adapter event IDs should use source-specific prefixes.
- **Status:** accepted for this ticket.

### Decision: attachments are references, not blob storage

- **Context:** Agent transcripts may include images, files, or large generated content. Storing all blobs in `.minitrace.json` would create privacy and size problems.
- **Options considered:** inline content, external object store, reference-only metadata.
- **Decision:** Store reference metadata plus bounded previews.
- **Rationale:** This keeps archives lightweight, privacy-safer, and easy to query. Full artifact storage can be layered later with `content_ref`/hash conventions.
- **Consequences:** Consumers may need access to the original path/URL/ref for full content. Tests should assert no large blobs are introduced unintentionally.
- **Status:** accepted for this ticket.

### Decision: keep annotations for review notes

- **Context:** Previous adapters sometimes used annotations to preserve source data because no better structure existed.
- **Options considered:** keep using annotations, migrate all source records to events/attachments, duplicate source records into both.
- **Decision:** New mappings should prefer events/attachments for source data; annotations remain for derived/human observations.
- **Rationale:** This makes downstream analysis semantically cleaner.
- **Consequences:** Some existing tests that expected attachment annotations may need updating. If backwards compatibility is necessary, adapters can temporarily keep compatibility annotations, but that should be explicit.
- **Status:** proposed; validate with tests and docs during implementation.

## Implementation phases

### Phase 1: schema and builders

Files:

- `pkg/minitrace/schema.go`
- `pkg/minitrace/builders.go`

Steps:

1. Add `Events` and `Attachments` to `Session`.
2. Add `Event` and `Attachment` structs.
3. Initialize empty slices in `BuildSessionSkeleton`.
4. Add `BuildEvent` and `BuildAttachment` helpers.
5. Run package tests to ensure existing JSON decode paths still work.

### Phase 2: SQLite materialization

Files:

- `pkg/minitracedb/schema.go`
- `pkg/minitracedb/materialize.go`
- `pkg/minitracedb/materialize_test.go`
- `pkg/minitracedb/schema_test.go`

Steps:

1. Register a new `attachmentsTable()` in `Tables()`.
2. Extend `eventsTable()` with `timestamp`, `attachment_id`, and `framework_metadata_json`.
3. Add indexes for attachment kind/tool/event lookups.
4. Add `insertExplicitEvent` and `insertAttachment` helpers.
5. Insert explicit events and attachments in `MaterializeSession`.
6. Update counts in materialization tests.
7. Add direct row assertions for explicit events and attachments.

### Phase 3: validation and docs

Files:

- `pkg/validate/json.go`
- `pkg/validate/json_test.go`
- `pkg/doc/query.md`
- `pkg/doc/writing-duckdb-queries.md`
- `pkg/doc/adapter-reference.md`

Steps:

1. Validate that `events` and `attachments` are either absent, `null`, or arrays.
2. Validate required fields for objects: `events[].id`, `events[].kind`, `attachments[].id`, `attachments[].kind`.
3. Add tests for null/empty/malformed arrays.
4. Update docs with JSON and SQLite query examples.

### Phase 4: adapter mappings

Files:

- `pkg/adapters/pi/convert.go`
- `pkg/adapters/pi/convert_test.go`
- `pkg/adapters/claudecode/convert.go`
- `pkg/adapters/claudecode/convert_test.go`
- `pkg/adapters/codex/convert.go`
- `pkg/adapters/codex/convert_test.go`

Steps:

1. Add event/attachment accumulators beside `turns`, `toolCalls`, and `annotations`.
2. Use helpers to build stable IDs.
3. Keep summaries privacy-safe.
4. Assign `session.Events` and `session.Attachments` before return.
5. Update existing tests and add minimized fixtures.

### Phase 5: preview and Goja/API visibility

Files:

- `pkg/minitracejs/import_builder.go`
- `pkg/minitracejs/import_builder_test.go`
- `cmd/go-minitrace/cmds/preview/session.go`
- `cmd/go-minitrace/cmds/preview/session_test.go`
- `pkg/doc/js-api-reference.md`

Steps:

1. Add event/attachment counts to preview structures.
2. Add sample rows controlled by existing privacy/sample options.
3. Ensure default previews do not print raw attachment content or raw source records.
4. Update JS API reference.

## Testing strategy

Use focused tests after each phase:

```bash
go test ./pkg/minitrace ./pkg/minitracedb -count=1
go test ./pkg/validate -count=1
go test ./pkg/adapters/pi ./pkg/adapters/claudecode ./pkg/adapters/codex -count=1
go test ./pkg/minitracejs/... ./cmd/go-minitrace/... -count=1
```

Before handoff, run the broader session-import relevant set:

```bash
go test ./pkg/adapters/... ./pkg/minitracedb ./pkg/minitracejs/... ./cmd/go-minitrace/... -count=1
```

For behavior validation, create or reuse small fixtures that include:

- Pi `session_info`, `custom`, `compaction`, `model_change`, and `thinking_level_change` records.
- Claude `attachment`, `mode`, `permission-mode`, and `ai-title` records.
- Codex `view_image`, `spawn_agent`, and `wait_agent` records.

## Query examples

Explicit event timeline:

```sql
SELECT event_id, timestamp, kind, title, summary
FROM events
WHERE session_id = ?
ORDER BY COALESCE(turn_index, 999999), COALESCE(ordinal, 999999), event_id;
```

Attachments by kind:

```sql
SELECT attachment_id, kind, media_type, name, path, tool_call_id
FROM attachments
WHERE kind = 'image'
ORDER BY timestamp, attachment_id;
```

Codex image views linked to tool calls:

```sql
SELECT a.attachment_id, a.path, t.tool_name, t.operation_type, t.result
FROM attachments a
LEFT JOIN tool_calls t
  ON t.session_id = a.session_id
 AND t.tool_call_id = a.tool_call_id
WHERE a.kind = 'image';
```

## Risks and mitigations

- **Schema drift:** Adding columns to `events` changes normalized SQL expectations. Mitigate with schema tests and docs updates.
- **ID collisions:** Explicit event IDs could collide with derived IDs. Mitigate by source prefixes such as `pi-compaction-000001`, `claude-attachment-000001`, `codex-rate-limits-000001`.
- **Privacy leaks:** Attachments may include paths, URLs, or content previews. Mitigate by bounding `TextPreview`, avoiding blob storage, and keeping preview defaults structural.
- **Annotation compatibility:** Some users may expect Claude attachments as annotations. Decide during implementation whether to keep a transition annotation or update tests/docs to reflect the cleaner semantics.
- **Frontend mismatch:** The web UI/protobuf layer may not render attachments immediately. Mitigate by documenting backend-first scope and exposing SQL/preview access first.

## Implementation status after first pass

The first implementation pass completed the backend and preview scope described in this guide:

- `Session.Events` and `Session.Attachments` exist in the canonical Go schema.
- The normalized SQLite schema is now `normalized-sqlite-v2` and includes an `attachments` table.
- The `events` table now accepts explicit source-event rows in addition to derived turn/tool/annotation events.
- Native JSON validation accepts absent/null/empty event and attachment arrays and requires `id`/`kind` for present objects.
- Pi maps `session_info`, `custom`, `compaction`, `model_change`, and `thinking_level_change` to events.
- Claude Code maps `mode`, `permission-mode`, `ai-title`, and `attachment` records to events/attachments.
- Codex derives `image_view`, `subagent_spawn`, `subagent_wait`, and `rate_limits` events plus image attachments from normalized tool/session metadata.
- Goja and CLI preview output includes event/attachment counts, kind breakdowns, and privacy-aware samples.

Known follow-up candidates:

- Preserve exact timestamps for Codex `rate_limits` events instead of deriving them from latest metadata only.
- Decide whether frontend/protobuf timeline rendering should expose events and attachments directly.
- Consider stricter validation once adapter-produced event/attachment shapes settle.

## References

- `pkg/minitrace/schema.go`: canonical session, turn, tool-call, annotation, event, attachment, and metrics structs.
- `pkg/minitrace/builders.go`: default constructors for sessions, turns, tool calls, and annotations.
- `pkg/minitracedb/schema.go`: normalized SQLite table descriptors, including existing `events` table.
- `pkg/minitracedb/materialize.go`: JSON-to-SQL materialization flow and derived timeline event insertion.
- `pkg/minitracedb/convert.go`: native/JSONL auto-loader and adapter dispatch.
- `pkg/adapters/pi/convert.go`: Pi JSONL conversion and current model/thinking/session handling.
- `pkg/adapters/claudecode/convert.go`: Claude Code JSONL conversion, metadata preservation, and current attachment annotation handling.
- `pkg/adapters/codex/convert.go`: Codex JSONL conversion, latest tool semantics, view-image and subagent tool mappings.
- `pkg/minitracejs/import_builder.go`: Goja/xgoja session preview API added by the preceding session-import ticket.
