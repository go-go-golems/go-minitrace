# Pi Transcript → Minitrace: Fields Lost in Conversion

## Critical Bug: `isError` Not Mapped to `output.success`

**Severity: HIGH**

The raw pi transcript has `isError: true` on toolResult messages that failed.
The minitrace converter maps this to `output.success: true` regardless — **inverting
the failure signal.**

### Raw (pi JSONL):
```json
{
  "role": "toolResult",
  "toolCallId": "call_function_gwqx815atsai_1",
  "toolName": "edit",
  "isError": true,
  "content": [{"type": "text", "text": "File not found: /path/to/file"}],
  "details": {}
}
```

### Minitrace output:
```json
{
  "output": {
    "success": true,
    "result": "File not found: /path/to/file",
    "error": null
  }
}
```

**Expected:**
```json
{
  "output": {
    "success": false,
    "error": "File not found: /path/to/file",
    "result": null
  }
}
```

This means **all queries that rely on `output.success` are wrong** for pi sessions.
Failed tool calls appear successful. The `reconstruct_files.py` script had to work
around this by parsing the result text for error patterns.

---

## Lost Fields: toolResult `details`

**Severity: MEDIUM**

Pi's edit tool returns a unified diff in the `details` field:

```json
{
  "details": {
    "diff": "    ...\n  9 | Component | Details |\n-13 | Alternatives | Separate VM or LXC |\n+13 | Terraform | Yes - ...",
    "firstChangedLine": 13
  }
}
```

This is **not captured** in minitrace. The diff is the most useful artifact for
understanding what an edit changed, and it's lost.

### Minitrace should add:
- `output.details.diff` — the unified diff from edit operations
- `output.details.first_changed_line` — line number of first change

---

## Lost Fields: Assistant Message Metadata

**Severity: MEDIUM**

### `stopReason`
Pi tracks why a response ended: `"toolUse"`, `"stop"`, `"error"`, `"aborted"`.
Minitrace only captures turn content, not the stop reason.

**Value for analysis:** Distinguishing model errors and aborts from normal stops.
In this session there were **14 aborted/error responses** from MiniMax-M2.7
(overloaded errors, user-initiated aborts).

### `errorMessage`  
Full error message when `stopReason` is `"error"` or `"aborted"`:
```json
{
  "stopReason": "error",
  "errorMessage": "529 {\"type\":\"error\",\"error\":{\"type\":\"overloaded_error\",\"message\":\"The server cluster is currently under high load...\"}}"
}
```

**Value:** Root-causing model failures, rate limiting, context window overflows.

### `responseId`
API provider's response ID (e.g., `"062f7701b15631351a5a1156fabd2740"`).
Useful for correlating with provider logs.

### `usage` (per-turn)
Per-turn token usage is captured in minitrace turns, but the raw format has
additional fields like `cost`:
```json
{
  "usage": {
    "input": 3392,
    "output": 789,
    "cacheRead": 9379,
    "cacheWrite": 0,
    "totalTokens": 13560,
    "cost": {"input": 0.001, "output": 0.001, "cacheRead": 0.001, "cacheWrite": 0, "total": 0.003}
  }
}
```

### `api` / `provider`
Which API was used: `"anthropic-messages"`, `"openai-completions"`, etc.
And provider: `"minimax"`, `"anthropic"`, `"openai"`, etc.

---

## Lost Structure: `parentId` Chain

**Severity: LOW (but useful for debugging)**

Every message has a `parentId` forming a DAG. Minitrace flattens this into
linear `turns[]` and `tool_calls[]` arrays. The parent chain is useful for:
- Reconstructing parallel tool call groups
- Understanding which tool results fed into which assistant decisions
- Debugging race conditions

---

## Lost Events: Non-Message Types

**Severity: LOW**

The raw transcript has event types beyond `message`:

| Event Type | Count | Description | In Minitrace? |
|------------|-------|-------------|----------------|
| `session` | 1 | Session header with `cwd` | Partial (operational_context) |
| `model_change` | 3 | Model switches mid-session | Partial (metrics.model_switches) |
| `thinking_level_change` | 1 | Thinking level (none/low/medium/high) | No |
| `compaction` | 1 | Context compaction with summary | No |
| `message` | 1176 | All conversation turns | Yes |

### `compaction` Event
```json
{
  "type": "compaction",
  "summary": "...(6369 chars of context summary)...",
  "firstKeptEntryId": "c49f2c63",
  "tokensBefore": 196709,
  "details": {
    "readFiles": ["path/to/file.md", ...],
    "modifiedFiles": ["path/to/file.yaml", ...]
  },
  "fromHook": false
}
```

This captures **what files were read/modified before compaction** — extremely
valuable for understanding file access patterns that led to data loss.

---

## Summary Table

| Field | Raw Location | Minitrace? | Impact |
|-------|-------------|------------|--------|
| `isError` | toolResult | ❌ **Always true** | **HIGH** — breaks all failure analysis |
| `details.diff` | toolResult | ❌ Lost | **MEDIUM** — lost edit diffs |
| `details.firstChangedLine` | toolResult | ❌ Lost | LOW |
| `stopReason` | assistant msg | ❌ Lost | **MEDIUM** — can't distinguish aborts |
| `errorMessage` | assistant msg | ❌ Lost | **MEDIUM** — lost model error details |
| `responseId` | assistant msg | ❌ Lost | LOW |
| `usage.cost` | assistant msg | ❌ Lost | LOW |
| `api` / `provider` | assistant msg | Partial | LOW |
| `parentId` chain | all messages | ❌ Lost | LOW |
| `compaction` events | top-level | ❌ Lost | **MEDIUM** — lost file access log |
| `thinkingLevel` | event | ❌ Lost | LOW |
| `session.cwd` | header | ✅ operational_context | — |

## Recommended Fixes (Priority Order)

1. **Map `isError` to `output.success = !isError`** — this is a correctness bug
2. **When `isError=true`, put result text in `output.error` not `output.result`**
3. **Capture `details.diff` in tool call output**
4. **Capture `stopReason` and `errorMessage` on assistant turns**
5. **Capture `compaction` events** with file lists
