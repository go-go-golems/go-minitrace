---
type: design
title: Pi Conversion and Web UI Rendering Gaps
status: active
intent: implementation-guide
topics:
  - pi
  - web-ui
  - conversion
  - minitrace
  - rendering
created: "2026-04-07"
---

# Pi Conversion and Web UI Rendering Gaps

This document catalogs all known issues with the Pi → minitrace conversion pipeline and the web UI's rendering of the resulting data. Each issue has: what's wrong, where the data lives, what needs to change, and in which files.

## Data Flow Overview

```
Pi JSONL ──→ pkg/adapters/pi/convert.go ──→ minitrace JSON
                                                   │
                                                   ▼
                                    cmd/.../serve/handlers_sessions.go
                                      (normalizeTurn / normalizeToolCall)
                                                   │
                                                   ▼
                                          HTTP JSON API response
                                                   │
                                                   ▼
                                     web/src/components/TranscriptViewer/
                                       (BlockBody.tsx / ToolCallRow.tsx)
```

Three layers where things can go wrong:
1. **Converter** (`pkg/adapters/pi/convert.go`) — Pi JSONL → minitrace JSON
2. **Server** (`cmd/.../serve/handlers_sessions.go`) — minitrace JSON → API response
3. **Frontend** (`web/src/`) — API response → rendered HTML

---

## Issue 1: Tool result messages rendered as "user" turns [FIXED]

**Layer:** Converter
**Status:** Fixed in this session

Tool result messages (`role: "toolResult"` in Pi JSONL) were converted to minitrace turns with `role: "user"`. This caused 737 tool results (file contents, bash output, skill files) to appear as fake "user" messages in the web UI.

### What was wrong

In `pkg/adapters/pi/convert.go`, `classifyTurnRole()` mapped `"toolResult"` → `("framework", "user")`:

```go
// pkg/adapters/pi/convert.go:437
func classifyTurnRole(role string) (*string, string) {
    switch role {
    case "user":      return ptr("human"), "user"
    case "assistant": return ptr("model"), "assistant"
    case "toolResult": return ptr("framework"), "user"  // ← BUG
    default:          return ptr("framework"), "user"
    }
}
```

### Fix applied

Added `continue` after `applyToolResult` for `role == "toolResult"` messages (line ~176). Tool results are already captured on the `tool_calls` array via `applyToolResult()`. No separate turn is needed — same approach as the Claude Code adapter.

**Result:** 1391 turns → 654 turns. Only `user` (human) and `assistant` (model) remain.

---

## Issue 2: Tool call summary shows "read read" instead of file path [FIXED]

**Layer:** Frontend
**Status:** Fixed in this session

### What was wrong

In `web/src/components/TranscriptViewer/ToolCallRow.tsx`, the summary label fell through to `tc.tool_name` when `command` was null:

```typescript
const cmd =
    tc.input.command ||
    tc.input.arguments?.cmd?.toString() ||
    tc.tool_name;  // ← fallback for read/write/edit/web_search
```

For `read` calls, `command` is null and `arguments.cmd` doesn't exist → shows `"read"`.

### Data in minitrace JSON

```json
{
  "tool_name": "read",
  "input": {
    "file_path": "~/.pi/agent/skills/docmgr/SKILL.md",
    "arguments": { "path": "/home/manuel/.pi/agent/skills/docmgr/SKILL.md" }
  }
}
```

### Fix applied

Added `file_path` and `query` to the fallback chain:

```typescript
const cmd =
    tc.input.command ||
    tc.input.file_path ||
    (tc.input.arguments?.query ? `"${tc.input.arguments.query}"` : undefined) ||
    tc.input.arguments?.cmd?.toString() ||
    tc.input.arguments?.path?.toString() ||
    tc.input.arguments?.url?.toString() ||
    tc.tool_name;
```

This covers: `bash` → command, `read/write/edit` → file_path, `web_search` → query, generic → path.

---

## Issue 3: Assistant content rendered as plain text, not markdown [FIXED]

**Layer:** Frontend
**Status:** Fixed in this session

### What was wrong

`BlockBody.tsx` rendered all turn content via `<Typography>` with `whiteSpace: "pre-wrap"`. Markdown formatting (headers, bullet lists, code blocks, bold) was displayed as raw text.

### Fix applied

Added `react-markdown` dependency. Assistant turns now render via `<Markdown>`, user turns stay as plain text (pre-wrap).

---

## Issue 4: Thinking traces not surfaced in web UI [NOT FIXED]

**Layer:** Server + Frontend

### What exists in the data

The Pi converter correctly extracts thinking content from Pi JSONL `thinking` blocks and stores them on the `Turn.Thinking` field:

```json
{
  "index": 1,
  "role": "assistant",
  "thinking": "Let me break down this complex request:\n\n1. Create a docmgr ticket...",
  "content": "I'll start by loading the relevant skills...",
  "model": "glm-5.1"
}
```

150 out of 654 turns have non-null `thinking`.

### Where it's lost

**Server:** `TurnResponse` in `handlers_sessions.go:87` does not include `thinking`:

```go
type TurnResponse struct {
    Idx             int                `json:"idx"`
    Role            string             `json:"role"`
    Source          string             `json:"source"`
    Content         string             `json:"content"`
    Timestamp       string             `json:"timestamp"`
    ToolCallsInTurn []ToolCallResponse `json:"tool_calls_in_turn"`
    // Missing: Thinking, Model, Usage
}
```

`normalizeTurn()` (line 337) drops `Thinking`, `Model`, `Usage` fields.

### What needs to change

1. **Server (`handlers_sessions.go`)**: Add fields to `TurnResponse`:

```go
type TurnResponse struct {
    // ... existing fields ...
    Thinking *string          `json:"thinking,omitempty"`
    Model    *string          `json:"model,omitempty"`
    Usage    *TurnUsageResponse `json:"usage,omitempty"`
}

type TurnUsageResponse struct {
    InputTokens     *int `json:"input_tokens,omitempty"`
    OutputTokens    *int `json:"output_tokens,omitempty"`
    CacheReadTokens *int `json:"cache_read_tokens,omitempty"`
    ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
}
```

2. **`normalizeTurn()`**: Wire the new fields:

```go
return TurnResponse{
    // ... existing ...
    Thinking: turn.Thinking,
    Model:    turn.Model,
    Usage:    normalizeUsage(turn.Usage),
}
```

3. **Frontend (`session.ts`)**: Add to `Turn` interface:

```typescript
export interface Turn {
  // ... existing ...
  thinking?: string | null;
  model?: string | null;
  usage?: {
    input_tokens?: number;
    output_tokens?: number;
    cache_read_tokens?: number;
    reasoning_tokens?: number;
  };
}
```

4. **Frontend (`BlockBody.tsx`)**: Render thinking as a collapsible block (like `<details>`) below the turn content. Show model as a chip badge.

---

## Issue 5: Edit tool calls don't show the diff [NOT FIXED]

**Layer:** Frontend

### What exists in the data

The Pi converter stores the full `edits` array in `input.arguments`:

```json
{
  "tool_name": "edit",
  "input": {
    "file_path": "~/code/wesen/.../InkSurface.h",
    "arguments": {
      "path": "/home/manuel/.../InkSurface.h",
      "edits": [
        {
          "oldText": "#include <QPointer>\n#include <QQuickPaintedItem>",
          "newText": "#include <QPointer>\n#include <QQuickItem>\n#include <QQuickPaintedItem>"
        }
      ]
    }
  }
}
```

The data is there in the API response (`input.arguments.edits`), but the frontend's expanded view only shows the `cmd` summary and `output.result`.

### What the frontend currently renders

In `ToolCallRow.tsx`, the expanded detail section shows:
- **Command** label → the summary string (just the file path for edits)
- **Output** → the success message "Successfully replaced 1 block(s)..."

### What it should render

For `edit` tool calls, the expanded view should show a **unified diff** for each edit in `arguments.edits[]`:

```
--- InkSurface.h
+++ InkSurface.h
@@ -1,2 +1,3 @@
 #include <QPointer>
+#include <QQuickItem>
 #include <QQuickPaintedItem>
```

### What needs to change

**Frontend (`ToolCallRow.tsx`)**: Detect edit tool calls and render diffs:

```typescript
// Pseudocode for the expanded section
if (tc.tool_name === "edit" && tc.input.arguments?.edits) {
  for each edit in tc.input.arguments.edits:
    render <UnifiedDiff oldText={edit.oldText} newText={edit.newText} />
}
```

Could use a lightweight diff library (e.g., `diff` npm package) to produce unified diff output from `oldText`/`newText` pairs. Or since it's exact text replacement, just color-code the removed/added lines directly (red for oldText lines, green for newText lines).

### Where in the code

`web/src/components/TranscriptViewer/ToolCallRow.tsx`, the `<Collapse in={expanded}>` section (around line 140). Currently only has Command/Output/Error blocks. Need to add an Edits/Diff section.

---

## Issue 6: Write tool calls don't show the content that was written [NOT FIXED]

**Layer:** Frontend

### What exists in the data

Write tool calls have the full file content in `input.arguments.content`:

```json
{
  "tool_name": "write",
  "input": {
    "file_path": "~/code/wesen/.../01-diary.md",
    "arguments": {
      "path": "/home/manuel/.../01-diary.md",
      "content": "---\ntype: reference\ntitle: Diary\n..."
    }
  }
}
```

### What it should render

For `write` tool calls, the expanded view should show the content in a scrollable code block:

```
📝 /path/to/file.md (8793 bytes)
─────────────────────
(content rendered as code or markdown)
```

### What needs to change

**Frontend (`ToolCallRow.tsx`)**: Detect write tool calls and render content:

```typescript
if (tc.tool_name === "write" && tc.input.arguments?.content) {
  render <CodeBlock content={tc.input.arguments.content} maxHeight={300} />
}
```

---

## Issue 7: Model and usage info not shown in web UI [NOT FIXED]

**Layer:** Server + Frontend

### What exists in the data

Every turn has `model` and most have `usage`:

```json
{
  "index": 1,
  "role": "assistant",
  "model": "glm-5.1",
  "usage": {
    "input_tokens": 8411,
    "output_tokens": 459,
    "cache_read_tokens": 704
  }
}
```

654/654 turns have model, 596/654 have usage.

This data is useful for:
- Seeing which model generated each response (the Pi session switched between `gpt-5.4`, `glm-5.1`, `o3`, etc.)
- Tracking token consumption over time
- Identifying expensive turns

### What needs to change

Same as Issue 4 — server needs to pass these through, frontend needs to render them.

---

## Pi JSONL Schema Reference

For context, here's what the Pi JSONL looks like:

### Line types

```jsonl
{"type": "session", "version": 3, "id": "...", "timestamp": "...", "cwd": "..."}
{"type": "model_change", "provider": "...", "modelId": "...", "timestamp": "..."}
{"type": "thinking_level_change", "thinkingLevel": "high", "timestamp": "..."}
{"type": "message", "timestamp": "...", "message": { ... }}
```

### Message roles

```json
// User message
{"type": "message", "message": {
  "role": "user",
  "content": [{"type": "text", "text": "..."}]
}}

// Assistant message with thinking + tool calls
{"type": "message", "message": {
  "role": "assistant",
  "usage": {"input": 8411, "output": 459, "cacheRead": 704, "cost": {"total": 0.25}},
  "content": [
    {"type": "thinking", "thinking": "...", "token_count": 150},
    {"type": "text", "text": "..."},
    {"type": "toolCall", "id": "call_xxx", "name": "read", "arguments": {"path": "..."}}
  ]
}}

// Tool result (separate message)
{"type": "message", "message": {
  "role": "toolResult",
  "toolCallId": "call_xxx",
  "toolName": "read",
  "content": [{"type": "text", "text": "file contents..."}]
}}
```

### Tool call argument shapes

```
read:    { path: string }
write:   { path: string, content: string }
edit:    { path: string, edits: [{ oldText: string, newText: string }] }
bash:    { command: string }
web_search: { query: string }
```

---

## Minitrace JSON Schema (what the converter produces)

### Turn

```go
type Turn struct {
    Index             int            `json:"index"`
    Timestamp         *string        `json:"timestamp"`
    Role              string         `json:"role"`         // "user" | "assistant"
    Source            *string        `json:"source"`       // "human" | "model" | "framework"
    Model             *string        `json:"model"`        // ← available, not passed to API
    ContentType       *string        `json:"content_type"`
    InputChannel      *string        `json:"input_channel"`
    Content           string         `json:"content"`
    FrameworkMetadata any            `json:"framework_metadata"`
    ToolCallsInTurn   []string       `json:"tool_calls_in_turn"`
    Thinking          *string        `json:"thinking"`     // ← available, not passed to API
    IntentMarkers     *IntentMarkers `json:"intent_markers"`
    Streaming         Streaming      `json:"streaming"`
    Usage             *Usage         `json:"usage"`        // ← available, not passed to API
}
```

### ToolCall

```go
type ToolCallInput struct {
    FilePath  *string `json:"file_path"`
    Command   *string `json:"command"`
    Arguments any     `json:"arguments"`   // ← has edits/content, not rendered for edit/write
}
```

---

## File Map

### Converter
- `pkg/adapters/pi/convert.go` — Pi JSONL → minitrace conversion
- `pkg/adapters/pi/convert_test.go` — tests

### Server (API)
- `cmd/go-minitrace/cmds/serve/handlers_sessions.go` — `TurnResponse`, `normalizeTurn()`, `normalizeToolCall()`
- `cmd/go-minitrace/cmds/serve/blocks.go` — session block assembly

### Frontend
- `web/src/types/session.ts` — TypeScript types (`Turn`, `ToolCall`, etc.)
- `web/src/components/TranscriptViewer/BlockBody.tsx` — turn rendering (content, thinking, model badges)
- `web/src/components/TranscriptViewer/ToolCallRow.tsx` — tool call rendering (summary, expanded diff/content)

### Schema
- `pkg/minitrace/schema.go` — Go structs for Turn, ToolCall, Usage, Thinking
