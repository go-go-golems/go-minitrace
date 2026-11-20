---
Title: Session counts and typed native inspection implementation notes
Ticket: GMT-015
Status: active
DocType: reference
Topics:
  - minitrace
  - analysis
Summary: Focused implementation and Goja validation after approved design c889b1a.
---

# Session counts and typed native inspection

## Step 1: inspect the existing contracts

Implementation waited for design03 to be written and committed (`c889b1a`). Inspection located query recipes in `pkg/minitracejs/query_view_session.go`, generated declarations in `typescript.go`, and actual native JSON tags in `pkg/nativeevidence/inspect.go` and `reader.go`. Initial guessed filenames `query_builder.go`/`query_recipes.go` did not exist; `rg` identified the real owner. No product behavior was inferred from those failed filename reads.

The existing recipe convention binds the optional session filter twice and exposes Go-owned SQL/argument methods. Native request results do not contain `base` or `keep`, even though the resolver consumes those fields. Compaction metadata is under `metadata`, and logical parent is a sibling of `boundary`; these exact shapes informed declarations and examples.

## Step 2: preserve session coverage in tool counts

Added `.SessionToolCounts()` to the existing fluent recipe builder. It selects `session_id` and `tool_count` from a sessions-driven left join, counts `tc.tool_call_id`, and orders by session ID. Optional `.SessionID(id)` remains parameter-bound. It counts actual normalized tool rows, including orchestration rows when present, rather than copying aggregate metrics or claiming process successes.

`TestSessionToolCountsGoja` constructs three synthetic archives in non-sorted order, including one zero-tool session and deliberately false stored tool-count metrics. Through Goja it builds the recipe, executes it on an in-memory database, verifies all three identities/counts/order, filters the zero-tool session, and checks that a SQL-looking filter value remains a parameter rather than selecting all sessions.

## Step 3: publish native shapes and execute the projection

Replaced the broad `Inspect(): Record<string, unknown>` declaration with `NativeInspection` and explicit coordinate, diagnostic, blob, request, message-coordinate, compaction and scan interfaces. Exact `rawJSON` and provider-defined `value`/`metadata` are retained. Optional compaction properties match the Go JSON `omitempty` tags; request and scan statuses name the current emitted values. No speculative replay fields were introduced.

The same declaration edit corrects `queryOne()` to return an object or null and permits the supported `execute({sql,args})` object alongside built recipes. QueryResult array types remain arrays, coordinated with the separate runtime error-envelope fix.

The executable example in `pkg/minitracejs/testdata/native_compaction_projection.js` is reproduced in the maintained JS API reference. Its Goja test verifies boundary line/byte coordinates, logical parent, preserved metadata, summary coordinates, before/following messages and assistant, and explicitly absent fields at an EOF boundary. The example projects missing values to null for reports without claiming exact replay.

## Validation

Passed:

```sh
go test ./pkg/minitracejs \
  -run 'TestSessionToolCountsGoja|TestNativeCompactionProjectionExample|TestNativeEvidenceJS' \
  -count=1
```

The first focused test run passed (0.658s package time). Fixtures contain only synthetic records. No production transcript or model calls were used. No standalone TypeScript compiler check was performed in this subtask; declarations are generated through the existing spec module and the parent integration gate owns broader validation.

Primary changed files: `pkg/minitracejs/query_view_session.go`, `typescript.go`, `session_tool_counts_test.go`, `native_evidence_test.go`, `testdata/native_compaction_projection.js`, and `pkg/doc/js-api-reference.md`. No CLI or getting-started document changes were made by this subtask, and no commit was created.
