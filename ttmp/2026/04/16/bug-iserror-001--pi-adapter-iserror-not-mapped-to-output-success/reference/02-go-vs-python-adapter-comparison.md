# Adapter Comparison: Go vs Python — isError Bug

## TL;DR

**Both adapters have the same bug.** Pi puts ALL tool results as message-level
objects (role="toolResult"), but both adapters only check for toolResult
**inside content blocks**. Neither reads `isError` from the message level.

- **Go adapter**: Has two code paths. Path 1 (content blocks) reads isError correctly but is never hit. Path 2 (message-level) hardcodes `false`. → **59 failures show as success=true**
- **Python adapter**: Only has the content-block path. Message-level toolResults are treated as regular text turns. `isError` is never read at all. → **Same 59 failures, plus tool results appear as spurious "user" turns**

## Pi Transcript Structure (Critical Context)

Pi's transcript format puts tool results as **separate message-level objects**,
NOT inside assistant content blocks:

```jsonl
{"type":"message","message":{"role":"assistant","content":[{"type":"toolCall","id":"tc-1","name":"edit","arguments":{...}}]}}
{"type":"message","message":{"role":"toolResult","toolCallId":"tc-1","toolName":"edit","isError":true,"content":[{"type":"text","text":"File not found: ..."}],"details":{}}}
```

From the jellyfin session:
- **0** toolResults inside assistant content blocks
- **606** toolResults as separate message-level objects
- **59** of those have `isError: true`

Neither adapter's content-block `toolResult` path is ever exercised for Pi data.

## Go Adapter (`go-minitrace/pkg/adapters/pi/convert.go`)

### Path 1: Content-block toolResult — lines 161-166
```go
case "toolResult", "tool_result":
    // ...
    applyToolResult(&toolCalls[pendingIndex], block["content"],
        boolValue(firstNonNil(block["isError"], block["is_error"])),  // ✅ reads isError
        timestampPtr)
```
**Verdict:** Correct but **dead code** — Pi never puts toolResult inside content blocks.

### Path 2: Message-level toolResult — lines 172-176
```go
if role == "toolResult" {
    // ...
    applyToolResult(&toolCalls[pendingIndex], contentBlocks,
        false,       // ❌ hardcoded false
        timestampPtr)
}
```
**Verdict:** BUG — `isError` is available at `msg["isError"]` but ignored.

### Impact
- 59 failed tool calls show `success: true, error: null` in minitrace output
- All queries filtering on `output.success` are wrong
- The `applyToolResult` function itself is correct — the bug is in the caller

### Fix (one line)
```go
// Line 175, change:
applyToolResult(&toolCalls[pendingIndex], contentBlocks, false, timestampPtr)
// To:
isErr := boolValue(firstNonNil(msg["isError"], msg["is_error"]))
applyToolResult(&toolCalls[pendingIndex], contentBlocks, isErr, timestampPtr)
```

## Python Adapter (`minitrace/adapters/pi/minitrace-pi-adapter.py`)

### Content-block toolResult — lines 218-231
```python
elif btype == "toolResult":
    result_id = block.get("toolUseId") or block.get("tool_use_id")
    result_content = block.get("content", "")
    is_error = block.get("isError", False)  # ✅ reads isError from block

    if result_id and result_id in pending_tool_calls:
        tc = pending_tool_calls.pop(result_id)
        tc["output"]["success"] = not is_error  # ✅ correct
```
**Verdict:** Correct but **dead code** — same as Go, never hit.

### Message-level toolResult handling
The Python adapter has **NO message-level toolResult handling at all**.
When it encounters a message with `role: "toolResult"`:

1. It iterates content blocks looking for `type: "toolResult"` — doesn't find any
2. The content is just `[{"type": "text", "text": "..."}]` — treated as regular text
3. Falls through to the source classification:
   ```python
   elif role == "toolResult":
       source = "framework"
       role = "user"  # tool results are "user" role in conversation
   ```
4. Creates a **spurious "user" turn** with the tool result text as content
5. The pending tool call **never gets a result** — stays in `pending_tool_calls`
6. After processing all records, orphaned tool calls get: `success: false, error: "no tool result received"`

### Impact (Python adapter)
**Worse than Go:**
1. **All tool results are lost** — none are matched to tool calls
2. **606 spurious "user" turns** appear in the conversation (one per tool result)
3. **All tool calls appear as orphans** with `success: false, error: "no tool result received"`
4. `isError` is never consulted
5. The turn sequence is corrupted with phantom user turns

### Fix
Add message-level toolResult handling in the `convert_session` function,
after the content block loop:

```python
# After the content block loop, check for message-level toolResult
if role == "toolResult":
    tool_call_id = msg.get("toolCallId")
    if tool_call_id and tool_call_id in pending_tool_calls:
        tc = pending_tool_calls.pop(tool_call_id)
        result_text = "\n".join(text_parts)  # already extracted from content blocks
        is_error = msg.get("isError", False)
        truncated, full_bytes, full_hash = truncate_content(result_text)
        tc["output"]["result"] = truncated
        tc["output"]["success"] = not is_error
        if is_error:
            tc["output"]["error"] = str(result_text)[:500]
        if full_bytes:
            tc["output"]["truncated"] = True
            tc["output"]["full_bytes"] = full_bytes
            tc["output"]["full_hash"] = full_hash
    # Don't create a turn for tool results
    continue  # skip turn creation
```

Also need to handle orphaned tool calls after the loop (already done for Go).

## Comparison Table

| Aspect | Go Adapter | Python Adapter |
|--------|-----------|----------------|
| Content-block toolResult | ✅ Reads isError | ✅ Reads isError |
| Message-level toolResult | ⚠️ Handled but ignores isError | ❌ Not handled at all |
| isError from message level | ❌ Hardcoded false | ❌ Never read |
| Tool results matched | ✅ All matched | ❌ None matched |
| Spurious turns | No | Yes (606 fake user turns) |
| Tool calls show as orphans | No | Yes (all of them) |
| Fix complexity | 1 line | ~15 lines |
| `details.diff` captured | ❌ No | ❌ No |

## Additional Shared Issues

Both adapters also lose these fields from message-level toolResults:

| Lost Field | Go | Python |
|------------|----|--------|
| `details.diff` | ❌ | ❌ |
| `details.firstChangedLine` | ❌ | ❌ |
| `toolName` on result (for matching) | Uses `toolCallId` only | Uses `toolUseId` only |
| `stopReason` / `errorMessage` on assistant | ❌ | ❌ |
| `compaction` events | ❌ | ❌ |
| Per-turn cost | ❌ | ❌ |
| `parentId` chain | ❌ | ❌ |

## Test Fixture (validates both adapters)

```jsonl
{"type":"session","version":3,"id":"test-iserror-001","timestamp":"2026-04-16T00:00:00Z","cwd":"/tmp"}
{"type":"message","id":"m1","parentId":null,"timestamp":"2026-04-16T00:00:01Z","message":{"role":"user","content":[{"type":"text","text":"test"}]}}
{"type":"message","id":"m2","parentId":"m1","timestamp":"2026-04-16T00:00:02Z","message":{"role":"assistant","content":[{"type":"toolCall","id":"tc-success","name":"bash","arguments":{"command":"echo hello"}}]}}
{"type":"message","id":"m3","parentId":"m2","timestamp":"2026-04-16T00:00:03Z","message":{"role":"toolResult","toolCallId":"tc-success","toolName":"bash","isError":false,"content":[{"type":"text","text":"hello"}],"details":{}}}
{"type":"message","id":"m4","parentId":"m3","timestamp":"2026-04-16T00:00:04Z","message":{"role":"assistant","content":[{"type":"toolCall","id":"tc-fail","name":"edit","arguments":{"path":"/tmp/nonexistent.txt","edits":[{"oldText":"a","newText":"b"}]}}]}}
{"type":"message","id":"m5","parentId":"m4","timestamp":"2026-04-16T00:00:05Z","message":{"role":"toolResult","toolCallId":"tc-fail","toolName":"edit","isError":true,"content":[{"type":"text","text":"File not found: /tmp/nonexistent.txt"}],"details":{}}}
```

**Expected output:**
- `tc-success`: `output.success = true`, `output.result = "hello"`
- `tc-fail`: `output.success = false`, `output.error = "File not found: /tmp/nonexistent.txt"`
- Turns: 1 user + 2 assistant (no toolResult turns)

**Current Go output:**
- `tc-success`: `success = true` ✅
- `tc-fail`: `success = true` ❌ (should be false)
- Turns: correct

**Current Python output:**
- `tc-success`: `success = false, error = "no tool result received"` ❌
- `tc-fail`: `success = false, error = "no tool result received"` ❌
- Turns: 1 user + 2 assistant + 2 spurious user turns ❌
