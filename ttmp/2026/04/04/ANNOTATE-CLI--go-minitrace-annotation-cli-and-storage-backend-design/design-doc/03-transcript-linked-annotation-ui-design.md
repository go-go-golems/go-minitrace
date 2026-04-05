---
Title: Transcript-Linked Annotation UI Design
Ticket: ANNOTATE-CLI
Status: active
Topics:
    - minitrace
    - annotations
    - react
    - frontend
    - ux
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: cmd/go-minitrace/cmds/serve/handlers_annotations.go
      Note: |-
        API already accepts scope.type and target_id; no major backend schema changes required for v1.
        Backend request fields already support scope_type and target_id for turn/tool-call annotations
    - Path: web/src/components/TranscriptViewer/AnnotationPanel.tsx
      Note: |-
        Current annotation list/add form; will become the entry point for jump-to-transcript behavior.
        Annotation cards should become clickable jump targets and scope-aware form launcher
    - Path: web/src/components/TranscriptViewer/BlockCard.tsx
      Note: |-
        Renders turns and tool calls; needs turn/tool-call anchors and inline annotation markers.
        Needs turn anchors
    - Path: web/src/components/TranscriptViewer/TranscriptViewer.tsx
      Note: |-
        Owns the transcript/annotations tab state and is the integration point for navigation.
        Parent component that owns tab state and should coordinate annotation-driven navigation
    - Path: web/src/types/session.ts
      Note: |-
        Frontend types for Annotation, Turn, ToolCall, SessionBlock.
        Defines Annotation scope plus Turn and ToolCall ids used for linking
ExternalSources: []
Summary: 'Design for making annotations navigable in the web UI: clicking annotations jumps to transcript context, transcript turns/tool calls show inline markers, and users can create turn-scoped annotations in place.'
LastUpdated: 2026-04-04T22:50:00Z
WhatFor: Give a future implementer a concrete UX and state-management plan for transcript-linked annotations in the React frontend.
WhenToUse: Use when implementing or reviewing annotation navigation, inline transcript markers, or turn/tool-call scoped annotation UX.
---


# Transcript-Linked Annotation UI Design

## Executive Summary

The current annotation feature is functionally complete at the storage, CLI, API, and basic UI layers, but it is not yet **navigable**. Users can create and list annotations, but when they see an annotation card in the web UI, they cannot jump to the message, turn, or tool call that the annotation describes.

This document proposes the next UI iteration:

1. **Annotation cards become navigable**.
   - Clicking an annotation card switches the viewer to the **Transcript** tab.
   - If the annotation scope is `turn`, the UI scrolls to the matching turn.
   - If the annotation scope is `tool_call`, the UI scrolls to the tool call row.
   - If the annotation scope is `session`, the UI scrolls to the top and highlights the session header.

2. **Transcript blocks show inline annotation markers**.
   - Turns with annotations show chips/badges.
   - Tool calls with annotations show small markers directly beside the tool-call row.
   - A block header may optionally show an aggregate annotation count.

3. **Users can annotate in context**.
   - A future “Annotate this turn” or “Annotate this tool call” action appears beside transcript items.
   - The add form pre-fills `scope.type` and `target_id`.

The important architectural point is that the backend data model already supports almost everything needed:

- `annotation.scope.type` = `session | turn | tool_call`
- `annotation.scope.target_id` = the specific session ID, turn index, or tool-call ID

So this design is mostly a **frontend integration and UX problem**, not a schema redesign.

---

## Problem Statement

### Current behavior

Today the React frontend renders annotations in a separate tab:

- `TranscriptViewer.tsx` toggles between **Transcript** and **Annotations**.
- `AnnotationPanel.tsx` renders a flat list of annotation cards and an add form.
- `BlockCard.tsx` renders blocks, turns, and tool calls, but it knows nothing about annotations.

This means the current workflow is fragmented:

1. Open a session.
2. Switch to **Annotations**.
3. Read an annotation card.
4. Manually switch back to **Transcript**.
5. Manually search for the relevant turn or tool call.

This is slow, error-prone, and especially bad for long transcripts.

### Why this matters

Annotations are valuable only if users can quickly answer:

- “**What exactly is this annotation referring to?**”
- “**Show me the turn where this happened.**”
- “**Show me the tool call that triggered this failure.**”

Without transcript linking, annotations are just disconnected notes.

### Functional gap

The data model already stores:

```json
{
  "scope": {
    "type": "turn",
    "target_id": "15"
  }
}
```

or:

```json
{
  "scope": {
    "type": "tool_call",
    "target_id": "tc_abc123"
  }
}
```

But the frontend currently does not:

- display the scope in a user-friendly way,
- scroll to the referenced transcript element,
- highlight the referenced transcript element,
- render per-turn or per-tool-call annotation markers,
- create scoped annotations from transcript context.

---

## Proposed Solution

## Summary

Implement transcript-linked annotations in **three layers**:

1. **Navigation layer** — annotation cards can activate transcript view and jump to target.
2. **Presentation layer** — transcript shows inline markers for annotated turns and tool calls.
3. **Creation layer** — transcript UI can create scoped annotations directly on the relevant item.

The proposal is intentionally incremental:

- **Phase A (quick win)**: click annotation card → switch to transcript tab → scroll/highlight target.
- **Phase B**: add inline markers in transcript.
- **Phase C**: add “Annotate this turn/tool call” affordances.

### Non-goals for this phase

- No schema redesign.
- No new backend tables.
- No full comment-thread model.
- No multi-annotation sidebar synchronized with transcript selection.
- No cross-session annotation explorer redesign.

---

## Proposed UX

### 1. Annotation card → transcript jump

When the user clicks an annotation card:

- the viewer switches to the **Transcript** tab,
- the transcript scrolls to the referenced element,
- the target is temporarily highlighted,
- the annotation card remains visible only if the user switches back to the Annotations tab.

#### Behavior by scope

| Scope | Target | Behavior |
|---|---|---|
| `session` | session id | switch to Transcript tab, scroll to top/session header, flash highlight |
| `turn` | turn index as string | expand containing block if needed, scroll to turn row, flash highlight |
| `tool_call` | tool call id | expand containing block if needed, scroll to tool-call row, flash highlight |

### 2. Inline markers in transcript

Transcript items show annotation presence without leaving the transcript:

- **Turn row**: badge like `2 annotations` or category chips like `ai-failure`, `question`
- **Tool call row**: compact chip or icon badge
- **Block header**: optional aggregate count, e.g. `#3 · 2 annotations`

The transcript should answer: “where are the annotated moments?” at a glance.

### 3. Annotate in place

When reading a turn or tool call, the user should not have to switch tabs and manually restate the target.

Future UX:

- Turn header: `+ Annotate`
- Tool call row: `+ Annotate`
- Clicking this opens a small inline or anchored form
- The form pre-fills:
  - `scope.type`
  - `target_id`
  - `session_id`

This preserves the existing API but improves user ergonomics dramatically.

---

## ASCII Screenshots / Wireframes

## Current UI (today)

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ TranscriptViewer                                                          │
│ [Sessions]  019bb3f6  Here's a plugin experiment in the browser... [Query]│
├────────────────────────────────────────────────────────────────────────────┤
│ Started ...   Duration ...   41 turns   307 tool calls   codex            │
├────────────────────────────────────────────────────────────────────────────┤
│ [Transcript (12 blocks)] [Annotations]                                    │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  If Transcript tab:                                                       │
│    #1  user prompt...                                                     │
│      assistant turn #2                                                    │
│      tool call bash                                                       │
│      tool call read                                                       │
│                                                                            │
│  If Annotations tab:                                                      │
│    [ai-failure] Codex: 307 tool calls                                     │
│    [observation] Codex: second session                                    │
│                                                                            │
│  PROBLEM: nothing connects the annotation card to the transcript item.     │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

## Proposed UI: click annotation card → jump to transcript

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ TranscriptViewer                                                          │
│ [Sessions]  019bb3f6  Here's a plugin experiment in the browser... [Query]│
├────────────────────────────────────────────────────────────────────────────┤
│ [Transcript (12 blocks)] [Annotations]                                    │
├────────────────────────────────────────────────────────────────────────────┤
│ Annotations tab                                                           │
│                                                                            │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ [ai-failure] Codex: 307 tool calls                              [x] │  │
│  │ scope: session · target: 019bb3f6-...                                │  │
│  │ detail: Extremely high tool count for exploration                     │  │
│  │ tags: codex pattern                                                   │  │
│  │ [Go to transcript]                                                    │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘

(click)

┌────────────────────────────────────────────────────────────────────────────┐
│ TranscriptViewer                                                          │
├────────────────────────────────────────────────────────────────────────────┤
│ [Transcript (12 blocks)] [Annotations]                                    │
├────────────────────────────────────────────────────────────────────────────┤
│ Transcript tab                                                            │
│                                                                            │
│  >>> highlighted session header / top anchor <<<                          │
│                                                                            │
│  #1  user prompt...                                                       │
│  #2  assistant...                                                         │
│  #3  assistant...                                                         │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

## Proposed UI: turn-scoped annotation marker

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ Block #4                                                                  │
├────────────────────────────────────────────────────────────────────────────┤
│ USER  #15                                                                 │
│ Please inspect the auth flow and explain why it failed.                   │
│                                                                            │
│ ASSISTANT #16                                      [Annotate]             │
│ The failure appears to come from a missing header...                      │
│                                                                            │
│   [ai-failure] [question]                                                 │
│   ↳ 2 annotations on this turn                                            │
│                                                                            │
│   tool: read file                                                         │
│   tool: bash                                                              │
│   tool: http_request                                  [1 annotation]      │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

## Proposed UI: annotation-focused transcript navigation

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ Transcript tab                                                            │
├────────────────────────────────────────────────────────────────────────────┤
│ #4  user prompt...                                                        │
│                                                                            │
│ ASSISTANT #16  ◄──────────── highlighted for 2.5 seconds                  │
│ The failure appears to come from a missing header...                      │
│ [ai-failure] [question] [View annotations] [Annotate]                     │
│                                                                            │
│ tool: http_request    ◄──── if tool_call scope, scroll here instead       │
│ [1 annotation]                                                            │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## YAML DSL React Sketch

Below is a high-level UI sketch in a YAML DSL style. This is not meant to compile; it is intended to clarify component composition, state ownership, and event flow.

```yaml
component: TranscriptViewer
props:
  - session: SessionDetail
  - onBack: fn
  - onQuerySession: fn
state:
  activeTab: transcript | annotations
  focusedAnnotationId: string | null
  focusedTarget:
    scopeType: session | turn | tool_call | null
    targetId: string | null
    highlightUntil: number | null
children:
  - Header:
      left:
        - BackButton
        - SessionIdLabel
      center:
        - SessionTitle
      right:
        - QueryButton
  - SessionInfoBar
  - TabBar:
      tabs:
        - transcript
        - annotations
  - Content:
      when: activeTab == transcript
      render:
        component: TranscriptPane
        props:
          session: session
          focusedTarget: focusedTarget
          onCreateScopedAnnotation: openAnnotationComposer
          onRevealAnnotationsForTarget: switchToAnnotationsAndFilter
      else:
        component: AnnotationPanel
        props:
          sessionId: session.id
          annotations: annotations
          onClose: setActiveTab(transcript)
          onNavigateToAnnotationTarget: navigateToAnnotationTarget
          onCreateAnnotation: createAnnotation

functions:
  navigateToAnnotationTarget(annotation):
    - set activeTab = transcript
    - set focusedAnnotationId = annotation.id
    - set focusedTarget.scopeType = annotation.scope.type
    - set focusedTarget.targetId = annotation.scope.target_id
    - requestAnimationFrame(scrollToFocusedTarget)
    - set highlightUntil = now + 2500ms

  openAnnotationComposer(scopeType, targetId):
    - set activeTab = annotations
    - prefill form.scopeType = scopeType
    - prefill form.targetId = targetId

component: TranscriptPane
props:
  - session
  - focusedTarget
  - onCreateScopedAnnotation
children:
  - SessionTopAnchor:
      id: session-top
  - Blocks:
      foreach: session.blocks
      render:
        component: BlockCard
        props:
          block: block
          focusedTarget: focusedTarget
          annotationIndex: annotationIndexByTarget
          onCreateScopedAnnotation: onCreateScopedAnnotation

component: BlockCard
props:
  - block
  - focusedTarget
  - annotationIndex
children:
  - BlockHeader:
      right:
        - BlockAnnotationCountBadge
  - Turns:
      foreach: block.turns
      render:
        component: TurnRow
        props:
          turn: turn
          annotations: annotationIndex.turn[turn.idx]
          isFocused: focusedTarget.scopeType == turn && focusedTarget.targetId == String(turn.idx)
          onAnnotate: onCreateScopedAnnotation(turn, String(turn.idx))
  - ToolCalls:
      foreach: turn.tool_calls_in_turn
      render:
        component: ToolCallRow
        props:
          toolCall: tc
          annotations: annotationIndex.tool_call[tc.id]
          isFocused: focusedTarget.scopeType == tool_call && focusedTarget.targetId == tc.id
          onAnnotate: onCreateScopedAnnotation(tool_call, tc.id)

component: AnnotationPanel
props:
  - sessionId
  - onNavigateToAnnotationTarget
children:
  - AnnotationList:
      foreach: annotations
      render:
        component: AnnotationCard
        props:
          annotation: annotation
          clickable: true
          onClick: onNavigateToAnnotationTarget(annotation)
          secondaryActions:
            - GoToTranscript
            - Delete
  - AnnotationCreateForm:
      fields:
        - category
        - title
        - detail
        - tags
        - scopeType
        - targetId
```

### Notes on the sketch

- `TranscriptViewer` owns the **tab state** and the **focused target state**.
- `AnnotationPanel` does not do the scrolling itself; it emits an event upward.
- `BlockCard` and `ToolCallRow` remain mostly presentational, but they need stable DOM anchors and “focused” styling hooks.
- The transcript needs an `annotationIndexByTarget` derived structure for efficient rendering.

---

## Data and State Model

## Existing data model (already sufficient for v1)

Frontend `Annotation` type:

```ts
export interface Annotation {
  id: string;
  timestamp: string;
  annotator: string;
  scope: {
    type: "session" | "turn" | "tool_call";
    target_id: string;
  };
  content: {
    category: string;
    tags: string[];
    title: string;
    detail: string;
  };
  taxonomy_mappings: {
    minitrace: string[];
    mast: string[];
    toolemu: string[];
  };
  classification?: string;
}
```

Transcript entities already have stable identifiers:

- `Turn.idx` — numeric turn index
- `ToolCall.id` — stable tool-call id
- `SessionDetail.id` — stable session id

This means the UI can derive anchors as follows:

| Scope | Target ID source | Suggested DOM anchor |
|---|---|---|
| `session` | `session.id` | `id="session-top"` |
| `turn` | `String(turn.idx)` | `data-turn-idx="15"` |
| `tool_call` | `toolCall.id` | `data-tool-call-id="tc_abc123"` |

## Derived annotation indexes

To render inline markers efficiently, the UI should precompute:

```ts
interface AnnotationIndex {
  byId: Record<string, Annotation>;
  bySession: Record<string, Annotation[]>;
  byTurn: Record<string, Annotation[]>;      // key = turn idx string
  byToolCall: Record<string, Annotation[]>;  // key = tool call id
}
```

Pseudocode:

```ts
function buildAnnotationIndex(annotations: Annotation[]): AnnotationIndex {
  const index = {
    byId: {},
    bySession: {},
    byTurn: {},
    byToolCall: {},
  };

  for (const ann of annotations) {
    index.byId[ann.id] = ann;

    if (ann.scope.type === "session") {
      (index.bySession[ann.scope.target_id] ??= []).push(ann);
    }
    if (ann.scope.type === "turn") {
      (index.byTurn[ann.scope.target_id] ??= []).push(ann);
    }
    if (ann.scope.type === "tool_call") {
      (index.byToolCall[ann.scope.target_id] ??= []).push(ann);
    }
  }

  return index;
}
```

This can be built in `TranscriptViewer` or in a small hook, e.g. `useAnnotationIndex(annotations)`.

---

## Design Decisions

### 1. Keep navigation state in `TranscriptViewer`

**Decision:** `TranscriptViewer` owns the active tab and focused target state.

**Why:**
- It already owns the transcript/annotations toggle.
- Both `AnnotationPanel` and `BlockCard` are children.
- The parent can coordinate tab switching plus scrolling.

**Consequence:**
- `AnnotationPanel` gets a new callback prop like `onNavigateToTarget(annotation)`.
- `BlockCard` gets props for `focusedTarget` and inline annotation lists/indexes.

### 2. Use DOM anchors + `scrollIntoView`, not router state

**Decision:** Implement transcript navigation as in-page scrolling, not route changes.

**Why:**
- The user is already inside a single session detail view.
- Changing routes would add complexity with little benefit.
- `scrollIntoView({ behavior: "smooth", block: "center" })` is enough.

**Consequence:**
- Need stable `ref` or data attributes on turn and tool-call rows.
- Need expansion logic for collapsed blocks before scrolling.

### 3. Prefer minimal API changes for v1

**Decision:** Do not change the API contract for navigation in v1.

**Why:**
- `scope.type` and `target_id` already exist.
- The UI already receives `Turn.idx` and `ToolCall.id`.
- This is enough to render anchors and navigate locally.

**Consequence:**
- Backend change is optional, not required.
- Future API improvements can enrich annotation query payloads with display hints, but are not necessary now.

### 4. Add inline markers before building a complex side panel sync model

**Decision:** Implement local markers/chips first, not a fully synchronized “selected annotation inspector”.

**Why:**
- Markers are the highest signal-to-effort improvement.
- They make the transcript readable immediately.
- A more advanced “selected annotation + linked transcript + reverse selection” system can come later.

### 5. Support session/turn/tool-call uniformly in the UI

**Decision:** Even if most current annotations are session-scoped, the UI should be designed for all three scopes from the start.

**Why:**
- The data model already supports it.
- The current feature gap is precisely that scoped annotations are not surfaced.
- Retrofitting later would be harder if the UI assumes session-only annotations.

---

## Alternatives Considered

### Alternative A — Keep annotations in a separate tab only

**Description:** Leave the transcript unchanged. Just render more metadata in annotation cards.

**Pros:**
- Minimal code change
- No scrolling logic
- No state coordination

**Cons:**
- Does not solve the core user problem
- Still forces manual transcript search
- Wastes the value of `scope.target_id`

**Verdict:** Rejected. This would preserve the current usability gap.

### Alternative B — Make annotation cards open a modal transcript snippet

**Description:** Clicking an annotation opens a modal with the referenced turn/tool call snippet.

**Pros:**
- Avoids transcript scrolling complexity
- Keeps context visible in a focused overlay

**Cons:**
- Hard to show surrounding context
- Duplicates transcript rendering logic
- Modal nesting gets awkward for long tool outputs

**Verdict:** Rejected for v1. It solves navigation partially but duplicates UI.

### Alternative C — Add a dedicated “annotation explorer” sidebar synchronized with transcript selection

**Description:** Persistent split-pane showing transcript and annotations at once, synchronized both ways.

**Pros:**
- Richest UX
- Great for annotation-heavy workflows

**Cons:**
- Large redesign of the viewer layout
- More state management complexity
- Overkill for the immediate gap

**Verdict:** Good future direction, not the first increment.

---

## Implementation Plan

## Phase 1 — Click annotation card to jump to transcript (quick win)

### Changes

- `TranscriptViewer.tsx`
  - Add state: `focusedTarget`, `focusedAnnotationId`
  - Pass `onNavigateToTarget` into `AnnotationPanel`
  - On callback:
    - set active tab = `transcript`
    - store focused target
    - schedule scroll after render

- `AnnotationPanel.tsx`
  - Make `AnnotationCard` clickable
  - Show scope label in card footer
  - Add explicit secondary action: `Go to transcript`

- `BlockCard.tsx`
  - Add data attributes / refs for turn rows and tool-call rows
  - Add temporary highlight styles for focused row

### Pseudocode

```ts
function handleNavigateToAnnotationTarget(annotation: Annotation) {
  setView("transcript");
  setFocusedTarget({
    scopeType: annotation.scope.type,
    targetId: annotation.scope.target_id,
    annotationId: annotation.id,
    highlightedAt: Date.now(),
  });
}

useEffect(() => {
  if (view !== "transcript" || !focusedTarget) return;

  const el = findTargetElement(focusedTarget);
  if (!el) return;

  el.scrollIntoView({ behavior: "smooth", block: "center" });
  flashHighlight(el);
}, [view, focusedTarget]);
```

## Phase 2 — Inline markers in transcript

### Changes

- Build annotation index in `TranscriptViewer`
- Pass turn/tool-call annotation arrays to `BlockCard` / `ToolCallRow`
- Add chips/count markers next to turns and tool calls
- Add optional aggregate marker to block header

### Marker examples

- Turn with one annotation: `[ai-failure]`
- Turn with multiple categories: `[ai-failure] [question]`
- Tool call with annotations: `tool: bash   [1 annotation]`

## Phase 3 — In-context creation

### Changes

- Add `Annotate` action beside turn header and tool-call row
- Reuse current annotation create flow, but prefill:
  - `scope.type`
  - `target_id`
- Extend `AnnotationPanel` form to allow hidden or read-only scope controls when launched from transcript

### Pseudocode

```ts
function onAnnotateTurn(turnIdx: number) {
  setView("annotations");
  setDraftAnnotation({
    scopeType: "turn",
    targetId: String(turnIdx),
    category: "observation",
    title: "",
    detail: "",
  });
}
```

## Phase 4 — Optional polish

- Reverse navigation: clicking a turn marker switches to Annotations tab filtered to that turn
- Better scope labels in cards, e.g. `Turn #15`, `Tool call bash (tc_abc123)`
- Keyboard shortcut: `j` / `k` for next/previous annotation target
- Persist selected annotation in URL hash or local state

---

## API Notes / Backend Notes

## Existing API already enough

Current create API:

```http
POST /api/sessions/{id}/annotations
Content-Type: application/json

{
  "scope_type": "turn",
  "target_id": "15",
  "category": "ai-failure",
  "title": "Missing auth header",
  "detail": "The assistant called the API before reading auth docs",
  "annotator": "user"
}
```

Current update/list APIs already preserve scope information.

### Optional backend niceties (not required)

These could help the frontend later, but are not needed for the first implementation:

- API response field like `scope_label` (`"Turn #15"`, `"Tool call bash"`)
- Server-side annotation counts per block/turn/tool call
- Session detail endpoint including annotations inline (instead of separate fetch)

---

## Risks and Edge Cases

### 1. Collapsed blocks hide the target

If the target turn/tool call is inside a collapsed `BlockCard`, scroll will fail or land on an invisible row.

**Mitigation:**
- `TranscriptViewer` must know how to expand the containing block before scrolling.
- This may require lifting `expanded` state out of `BlockCard` or exposing an imperative expansion prop.

### 2. Turn target format mismatch

The annotation `target_id` is a string, but `Turn.idx` is a number.

**Mitigation:**
- Normalize to `String(turn.idx)` everywhere in the frontend.

### 3. Tool call rows may not expose stable DOM anchors yet

**Mitigation:**
- Add `data-tool-call-id={tc.id}` in `ToolCallRow`.

### 4. Session-scoped annotations have no precise location

**Mitigation:**
- For `scope: session`, scroll to the top header or to a dedicated “session-top” anchor.

### 5. Large transcripts + smooth scrolling

**Mitigation:**
- Keep navigation simple for v1.
- Do not build a virtualized list yet.

---

## Open Questions

1. **Should block expansion state be lifted from `BlockCard` to `TranscriptViewer`?**
   - Likely yes, if we want deterministic programmatic expansion before scrolling.

2. **Should the add form live in the AnnotationPanel only, or can it also appear inline in the transcript?**
   - v1 can keep the form in AnnotationPanel and prefill it from transcript actions.

3. **How should multiple annotations on the same turn be displayed?**
   - Chips for up to 2 categories + `+N` overflow is likely sufficient.

4. **Should clicking a turn marker switch to the Annotations tab, or open a small popover inline?**
   - Inline popover is nicer, but tab switch is simpler.

5. **Should session-scoped annotations be shown in transcript at all?**
   - Probably yes, as a header-level marker near the session title/info bar.

---

## References

- `design-doc/01-annotation-storage-backend-and-cli-design-decision.md`
- `design-doc/02-annotation-cli-implementation-guide.md`
- `reference/02-diary.md`
- `POSTMORTEM.md`
