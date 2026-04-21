---
Title: Bug Report — Pi adapter ignores isError on message-level toolResult
Ticket: bug-iserror-001
Status: active
Topics:
  - bug
  - pi-adapter
  - converter
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
  - /home/manuel/code/wesen/corporate-headquarters/go-minitrace/pkg/adapters/pi/convert.go
ExternalSources: []
Summary: >
  The Pi adapter's convert.go has two code paths for processing tool results.
  The message-level toolResult path (line 172-176) hardcodes isError=false,
  ignoring the isError field that Pi puts on message-level toolResult objects.
  This causes failed tool calls to appear as successful in minitrace output.
LastUpdated: 2026-04-16
---

# Bug Report: Pi adapter ignores `isError` on message-level toolResult

## Summary

**Severity:** High (correctness bug — inverts failure signal)  
**File:** `pkg/adapters/pi/convert.go`  
**Affected:** All Pi session conversions where tool calls fail  
**Impact:** Every DuckDB query using `output.success` returns wrong results for Pi sessions

## Root Cause

The Pi adapter has **two code paths** for processing tool results:

### Path 1: Content-block toolResult (inside assistant messages) — line 161-166 ✅

```go
case "toolResult", "tool_result":
    toolCallID := firstNonEmpty(stringValue(block["toolUseId"]), stringValue(block["use_use_id"]))
    if toolCallID != "" {
        if pendingIndex, ok := pendingToolCalls[toolCallID]; ok {
            applyToolResult(&toolCalls[pendingIndex], block["content"],
                boolValue(firstNonNil(block["isError"], block["is_error"])),  // ← CORRECT: reads isError
                timestampPtr)
            delete(pendingToolCalls, toolCallID)
        }
    }
```

### Path 2: Message-level toolResult — line 172-176 ❌

```go
if role == "toolResult" {
    toolCallID := stringValue(msg["toolCallId"])
    if toolCallID != "" {
        if pendingIndex, ok := pendingToolCalls[toolCallID]; ok {
            applyToolResult(&toolCalls[pendingIndex], contentBlocks,
                false,       // ← BUG: hardcoded false, ignores msg["isError"]
                timestampPtr)
            delete(pendingToolCalls, toolCallID)
        }
    }
}
```

## Why Both Paths Exist

Pi's transcript format has two ways tool results appear:

1. **Embedded in assistant content blocks** — some adapters (Claude, etc.) embed tool results as content blocks inside assistant messages
2. **As separate message-level objects** — Pi writes tool results as standalone messages with `role: "toolResult"` at the top level, with `isError` as a **message-level field**, not inside content blocks

## Evidence from Real Transcript

### Raw pi JSONL (message-level toolResult):

```json
{
  "type": "message",
  "id": "fd2bdb0e",
  "parentId": "5f360ea1",
  "timestamp": "2026-04-16T01:48:39.447Z",
  "message": {
    "role": "toolResult",
    "toolCallId": "call_function_gwqx815atsai_1",
    "toolName": "edit",
    "isError": true,
    "content": [{"type": "text", "text": "File not found: /path/to/01-diary.md"}],
    "details": {}
  }
}
```

### Minitrace output (incorrect):

```json
{
  "id": "call_function_gwqx815atsai_1",
  "tool_name": "edit",
  "output": {
    "success": true,       // ← WRONG: should be false
    "error": null,         // ← WRONG: should contain "File not found: ..."
    "result": "File not found: /path/to/01-diary.md"  // ← Should be null or omitted
  }
}
```

### Expected minitrace output:

```json
{
  "id": "call_function_gwqx815atsai_1",
  "tool_name": "edit",
  "output": {
    "success": false,
    "error": "File not found: /path/to/01-diary.md",
    "result": null
  }
}
```

## Affected Queries

This bug makes ALL failure-analysis queries return wrong results:

```sql
-- This returns ZERO rows for Pi sessions, even though 3+ tool calls failed
SELECT * FROM sessions_base, UNNEST(tool_calls) AS t(tc)
WHERE json_extract(tc, '$.output.success') = false;
```

The `reconstruct_files.py` script had to work around this by parsing result text:

```python
_FAILED_RESULT_PATTERNS = [
    "File not found",
    "No such file",
    "Permission denied",
]
```

## Fix

In `pkg/adapters/pi/convert.go`, line 175, change:

```go
// Before (buggy):
applyToolResult(&toolCalls[pendingIndex], contentBlocks, false, timestampPtr)

// After (fixed):
isErr := boolValue(firstNonNil(msg["isError"], msg["is_error"]))
applyToolResult(&toolCalls[pendingIndex], contentBlocks, isErr, timestampPtr)
```

## Additional Lost Fields in Message-Level Path

While fixing isError, also capture the `details` field that's lost on the message-level path:

### Raw:
```json
{
  "role": "toolResult",
  "toolName": "edit",
  "details": {
    "diff": "    ...\n  9 | Component | Details |\n-13 | Alternatives | ...\n+13 | Terraform | ...",
    "firstChangedLine": 13
  }
}
```

The `details` object contains the **unified diff** for edit operations — extremely
valuable for understanding what changed. This is currently discarded entirely.

### Suggested addition to minitrace schema:

Add to `ToolCallOutput`:

```go
type ToolCallOutput struct {
    // ... existing fields ...
    Details *ToolCallDetails `json:"details,omitempty"`
}

type ToolCallDetails struct {
    Diff            string `json:"diff,omitempty"`
    FirstChangedLine *int  `json:"first_changed_line,omitempty"`
}
```

## Test Case

Create a test fixture with a message-level toolResult where `isError=true`:

```jsonl
{"type":"session","version":3,"id":"test-001","timestamp":"2026-04-16T00:00:00Z","cwd":"/tmp"}
{"type":"message","id":"m1","parentId":null,"timestamp":"2026-04-16T00:00:01Z","message":{"role":"user","content":[{"type":"text","text":"test"}]}}
{"type":"message","id":"m2","parentId":"m1","timestamp":"2026-04-16T00:00:02Z","message":{"role":"assistant","content":[{"type":"toolCall","id":"tc-1","name":"edit","arguments":{"path":"/tmp/test.md","edits":[{"oldText":"a","newText":"b"}]}}]}}
{"type":"message","id":"m3","parentId":"m2","timestamp":"2026-04-16T00:00:03Z","message":{"role":"toolResult","toolCallId":"tc-1","toolName":"edit","isError":true,"content":[{"type":"text","text":"File not found: /tmp/test.md"}],"details":{}}}
```

**Expected:** `output.success = false`, `output.error = "File not found: /tmp/test.md"`  
**Current:** `output.success = true`, `output.error = null`

## Scope of Impact

From the jellyfin session alone:
- 3 failed edit calls with `isError=true` → all appear as `success=true`
- 0 queries could detect these failures via `output.success`
- The `reconstruct_files.py` script needed a heuristic workaround

This likely affects **every Pi session with any failed tool call** in every minitrace archive.

## Other Fields Lost in Conversion (same ticket, lower priority)

See `sources/pi-transcript-vs-minitrace-field-comparison.md` for the full field comparison.
