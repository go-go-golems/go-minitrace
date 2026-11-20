---
Title: Query fidelity implementation notes and regression evidence
Ticket: GMT-015
Status: active
DocType: reference
Topics: [analysis, minitrace]
---

## Step 1: Make incomplete query evidence observable

The prior audit's three-byte `abé` probe returned a replacement character without a truncation flag. This implementation keeps the existing SQLite runner, database builders and recipe objects, adds explicit completeness metadata, and provides an opt-in mode that refuses partial results. It does not change transcript normalization or claim that a complete SQL result proves the underlying transcript was complete.

### Prompt Context
**Delegated task summary:** Implement M01/M02/M13/M14 query fidelity, explicit byte-limit aliases, strict evidence through JS, and a safe recipe execution bridge. The parent owns the user-facing guide and CLI integration. This is a task summary, not a verbatim user quotation.
**Assistant interpretation:** Fix silent loss while preserving the explicitly requested old byte-limit alias and permissive display behavior.
**Inferred user intent:** Evidence extraction should not require repeated byte-length preflight SQL or manual recipe argument bridging.

### What I did
- Changed `pkg/minitracedb/query.go` to preserve whole UTF-8 code points within a byte budget, diagnose invalid input UTF-8, and report cell coordinates and byte lengths.
- Added preferred `QueryOptions.MaxCellBytes` and kept `MaxCellChars` as the explicitly requested deprecated byte-limit alias. A positive MaxCellBytes wins in Go options. Defaults remain4000 bytes.
- Added `StrictEvidence` to Go options, the JS DB builder and the JS limits builder.
- Added `db.execute(recipe)`, accepting current recipe objects with sql()/args() or plain `{sql, args}` values. Parameters remain bound through SQLite.
- Added dedicated Go and Goja fidelity regression tests and a synthetic repository-backed JS command under ticket scripts/query-fidelity.

### Exact result contract

`QueryResult` retains columns, rows, count, truncated and error. New fields are:

- `complete`: always emitted; false on errors, row overflow or lossy string/blob conversion.
- `rowsTruncated`: true only when the query has an additional row beyond MaxRows.
- `cellDiagnostics`: an entry per affected cell, with zero-based rowIndex and columnIndex, column name, originalBytes, emittedBytes, truncated and optional invalidUtf8.
- The existing `truncated` flag now covers row overflow **or any lossy cell conversion**. This deliberately closes the previous false-negative cell-clipping case.

For valid `abé` with a three-byte budget, permissive output is `ab`, originalBytes4, emittedBytes2, completefalse. The implementation does not insert a replacement character merely because a limit falls inside a rune. If the source itself contains invalid UTF-8, it repairs the text for JSON presentation, marks invalidUtf8, and sets completefalse even if the repaired text fits the limit. String/blob cells are the scope of cell diagnostics; this is not a binary-output transport.

Strict mode sets `error: "strict evidence: query result is incomplete"`, empties rows and sets count0 while retaining columns and diagnostics. `queryResult()` and `execute()` return this envelope; the existing `query()` and `queryOne()` interfaces turn its error into their normal Go/Goja error. Row overflow is subject to the same strict rule as cell clipping. Exact boundary results remain complete.

```javascript
const db = mt.db().RuntimeArchives().SQLiteMemory()
  .MaxCellBytes(16000).StrictEvidence().Build();
try {
  const result = db.execute(mt.query().ToolRows().SessionID(id).Build());
  if (result.error) throw new Error(result.error);
  // result.complete now covers query row/cell limits, not historical-source completeness.
  return result.rows;
} finally { db.close(); }
```

The equivalent limits facade is `mt.limits().CellBytes(16000).StrictEvidence().Build()`, passed to `.Limits(...)`. `.MaxCellChars(...)` and `.CellChars(...)` remain supported aliases. In the fluent setters, the last explicit cell-limit setter wins; applying a limits object chooses its positive MaxCellBytes over its legacy field. `StrictEvidence(false)` remains a deliberate opt-out.

### Why
- A query-result envelope can carry coordinates and original lengths without placing raw source bodies in diagnostics.
- Strict mode is opt-in so display callers can still render bounded previews.
- Recipe objects already provide SQL and arguments; execute composes those existing methods rather than defining a new interpreter.
- Unknown or malformed recipe shapes fail explicitly. SQL errors retain the current error-envelope behavior; this change does not silently redefine every error as an exception.

### What worked
- The database suite passed with ASCII, accented and supplementary-character boundaries, invalid UTF-8, BLOB text, oversized JSON, row-only overflow, combined row/cell overflow and exact-limit cases.
- The Goja suite passed with DB and limits-builder paths, legacy aliases, method-based and data-based recipes, bound arguments, JSON serialization, strict envelopes, and query/queryOne rejection.

### What didn't work
- The first JS test fixture expected12 bytes for an11-byte SQL-shaped argument. This was a fixture arithmetic error; corrected to11.
- The next suite run was briefly blocked by a concurrently authored provenance test treating row slices as QueryResult. Its owner corrected the test; no cross-owner edit was made.

### What I learned
- Valid UTF-8 clipping and invalid-source UTF-8 are different cases. Repairing invalid original bytes still loses evidence, so it must not be called complete.
- Strict-mode diagnostics refer to the zero-based candidate output coordinates, even though rows are withheld; count represents returned rows and is0.

### What was tricky to build
- Default resolution and fluent setters must preserve the old byte-limit alias without letting a resolved new default shadow a later legacy setter.
- A result marked incomplete must not be returned through the simpler rows-only strict interfaces. Their existing envelope-error handling provides that enforcement.

### What warrants a second pair of eyes
- Consumers that interpreted truncated as exclusively row overflow should now inspect rowsTruncated and cellDiagnostics. This widening is intentional.
- Complete concerns the returned SQL projection and its limits. A SQL LIMIT chosen by the author, missing native records, incomplete archives, normalization loss, and causal usefulness remain outside this flag's contract.
- Duplicate SQL column labels still use the established map-row representation; callers should use unique aliases. No unrelated row representation refactor was attempted.

### What should be done in the future
- Update all public help/examples to favor CellBytes and show strict envelope handling; the parent owns the broader guide/CLI changes.
- Consider a separate binary-cell API only if a real consumer needs byte-exact BLOB transport rather than text evidence.

### Code review instructions

```bash
go test ./pkg/minitracedb ./pkg/minitracejs
go run ./cmd/go-minitrace query commands \
  --query-repository ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/query-fidelity \
  fidelity-probe --output json
```

### Technical details

Owned implementation files: pkg/minitracedb/query.go, pkg/minitracejs/db_builder.go, pkg/minitracejs/builders.go. Dedicated tests: pkg/minitracedb/query_fidelity_test.go and pkg/minitracejs/query_fidelity_test.go. Changes were formatted with gofmt. No commit was made by this subtask.

## Step 2: Verify the repository-backed CLI boundary

The synthetic `fidelity-probe` command completed successfully through `go run ./cmd/go-minitrace query commands ... --output json`. Both the database-backed JS runtime and the outer CLI JSON serializer retained the new metadata. No historical archives, tokens, network calls or model execution were involved.

### Prompt Context
See Step 1. The implementation requirement includes demonstrable Go, Goja and CLI behavior, not only a helper-unit test.

### What I did
Ran the command recorded above and saved its scratch output to `/private/tmp/gmt-015-query-fidelity-probe.json`. Also checked the owned implementation diff with `git diff --check`.

### What worked
The permissive result has rows `[{"evidence":"ab"}]`, count1, completefalse, truncatedtrue, rowsTruncatedtrue, and a diagnostic identifying row0/column0, originalBytes4, emittedBytes2. The strict result carries the same completeness diagnostics, error `strict evidence: query result is incomplete`, rows `[]` and count0. Both are serialized in the command's result field without broken UTF-8.

### What didn't work
No additional failures in this boundary check.

### What I learned
The `execute` method can be used directly inside repository-backed JS commands with data-shaped bound recipes; callers do not need to unwrap recipe SQL and arguments manually.

### What was tricky to build
The CLI emits rows, so the probe intentionally wraps the complete result envelope as a JSON string column instead of assuming arbitrary nested command output shape.

### What warrants a second pair of eyes
The outer CLI's presentation row is separate from the inner evidence envelope. Real analysis commands should choose whether to emit the envelope or reject `error` before emitting result rows.

### What should be done in the future
The parent integration will run broader checks and document the CLI flags; no further local changes are needed for this probe.

### Code review instructions
Inspect the persisted probe command, run it as documented, and verify that the permissive and strict result envelopes differ in rows/error while preserving cell coordinates and byte lengths.

### Technical details
`go test ./pkg/minitracedb ./pkg/minitracejs` passed. `git diff --check` on the three owned implementation files passed. Formatting used gofmt; no format_file tool was available in this runtime.
