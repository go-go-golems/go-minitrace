---
Title: Query contract and malformed archive diagnostic implementation notes
Ticket: GMT-015
Status: active
Topics:
  - analysis
  - minitrace
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
  - Path: pkg/minitracedb/query.go
    Note: Central error envelope constructor
  - Path: pkg/minitracedb/convert.go
    Note: Conservative normalized input diagnostic recognition
  - Path: pkg/minitracejs/db_builder.go
    Note: Targeted execute string argument guidance
ExternalSources: []
Summary: Implemented and tested the narrow result-contract and input-diagnostic fixes motivated by persona traces.
LastUpdated: 2026-11-17T16:00:00-05:00
WhatFor: Retrace implementation decisions and negative-path validation independently of historical experiment artifacts.
WhenToUse: Review the query contract patch or extend its regression tests.
---

# Step 1: Inspect the existing paths before editing

The implementation began only after design03 was committed as c889b1a. AGENT.md was read. Source inspection used rg for QueryResult, recipe.sql, AutoConvert, and decode calls, then read query.go, convert.go, db_builder.go and the existing query/conversion/Goja fidelity tests. No historical model answers, generated commands, or experiment receipts were changed.

The array mismatch originated in QueryRunner.QueryResult: successful queries initialize Rows, but fourteen error returns used QueryResult{Error:...}. JSON therefore encoded nil Columns/Rows as null even though JS declarations promised arrays. The loader tried decodeNativeSession first but discarded any decode error before reporting AutoConvert disabled or attempting conversion. The execute adapter boxed arbitrary arguments into an object; a SQL string consequently failed with a generic recipe.sql shape message.

Owned implementation files:

- pkg/minitracedb/query.go
- pkg/minitracedb/convert.go
- pkg/minitracejs/db_builder.go
- pkg/minitracedb/negative_contract_test.go
- pkg/minitracejs/negative_contract_test.go

Other agents own CLI, declarations, recipes, native tests, and maintained API docs. Their concurrent edits appeared in the worktree diff but were not modified here.

# Step 2: Normalize error envelopes without changing error policy

A small unexported queryErrorResult(message) constructor returns Columns:[], Rows:[], Error:message, Count:0 and Complete:false. Every existing early query error path now uses it, including uninitialized DB, SQL validation, connection failure, authorization/preparation, SQL execution, column limit, scan, and iteration failures. This is a concrete returned-value contract; no custom JSON marshaler hides nil values only at serialization time.

Strict-evidence failures continue to retain available columns and cell/row diagnostics while clearing Rows and Count. Convenience query/queryOne still throw through their existing Go error conversion; queryResult/execute still return envelopes for query errors. Empty evidence on error remains distinguishable from a complete zero-row answer. This change does not make arbitrary manually constructed Go QueryResult{} values non-nil: it guarantees results emitted by the query runner. No TypeScript changes are needed for this contract fix because runtime behavior now matches the declared arrays.

# Step 3: Make execute(string) actionable

The adapter recognizes a primitive goja.String before boxing a recipe and raises:

```
execute requires a query recipe, not SQL text; use queryResult(sql, ...args) or execute({sql: sql, args: [...]})
```

Method recipes and plain {sql,args} recipes remain supported. The first implementation draft used value.Export().(string); source review caught that exporting an entire object could unnecessarily evaluate unrelated properties. Before final verification it was replaced with a goja.String type assertion. A regression test uses an unrelated throwing getter on an otherwise valid recipe to ensure only the recipe fields are accessed. This was a review improvement, not a test-failure repair.

The change deliberately does not add another execute overload. A boxed String object is still handled as a malformed recipe rather than being treated as SQL text. The targeted hint addresses the primitive string mistake observed in the pilot without coercing arbitrary inputs.

# Step 4: Preserve malformed normalized input errors conservatively

After decodeNativeSession fails, looksLikeNormalizedSession checks whether the entire payload is a JSON object containing id and turns keys plus at least one normalized companion key: schema_version, profile, provenance, tool_calls, or metrics. It first excludes records recognized by the existing native-format detector. A matching candidate returns `decode normalized minitrace session <source>: <original decode error>`, wrapping the original error with %w. Go callers can inspect json.UnmarshalTypeError through errors.As; JS receives the original field/type explanation.

For example, `{ "id":"broken", "turns":[], "metrics":[] }` now identifies metrics as an array that cannot decode into minitrace.Metrics, whether AutoConvert is true or false. Changing its filename is not suggested as a repair.

This is diagnostic classification after failed decoding, not a new parser, schema validator, or successful-input routing policy. Recognized Pi, role-only Pi, Codex, Claude Code, and Copilot records keep their conversion path even if additional fields resemble a malformed normalized archive. Multiple-record JSONL fails the whole-object shape check and stays on the existing JSONL path. Unknown JSON with only id and turns is not sufficient. Syntactically invalid JSON and documents missing these identifying keys retain existing fallback diagnostics; the implementation does not claim comprehensive malformed-input classification.

# Step 5: Validation and evidence

Focused negative tests were added and passed on their first execution:

```
go test ./pkg/minitracedb ./pkg/minitracejs -run 'Test(QueryErrorEnvelopes|MalformedNormalized|NormalizedErrorDetection|JSNegative)' -count=1
```

The full two affected packages then passed:

```
go test ./pkg/minitracedb ./pkg/minitracejs -count=1
```

After the goja.String review improvement and unrelated-getter regression, the same complete package command was rerun. Formatting used gofmt on the owned Go files; the requested format_file tool is not available in this environment.

Coverage includes:

- Empty SQL, writes, multiple statements, denied system tables, nonexistent columns, missing bound arguments, maximum-column rejection, a closed DB, and a nil runner.
- Direct Go slices and serialized JSON arrays, with Count:0 and Complete:false on errors.
- Goja Array.isArray checks on queryResult and execute errors, including JSON roundtrip.
- Existing query/queryOne exception behavior and successful bound recipe execution.
- SQL-string argument hint and absence of unrelated recipe getter evaluation.
- Typed normalized metrics-array errors with AutoConvert both off and on.
- Five native-format control records with misleading normalized-looking extra keys, all still converted successfully.

The constructor is used for scan/iteration failures, but tests do not inject artificial driver failures for every internal branch. Existing package suites cover successful conversion/query/fidelity behavior. This subtask makes no full-repository pass claim; root handles integrated validation and records unrelated baseline/environment limitations separately.

# Review and next steps

The code remains additive and localized. The classifier intentionally favors avoiding false native-format classification over catching every malformed normalized document. If broader malformed-input support is needed, add explicit input-kind metadata or schema validation as a separate design; do not grow filename-based heuristics from this single pilot error.

The new tests verify externally visible contract behavior rather than the spelling of the helper implementation. No commits were created by this subagent. Root should review and commit the exact owned files together with coordinated integration work.
