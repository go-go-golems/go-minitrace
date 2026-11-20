---
Title: Persona review followups result contracts discoverability and enforced test isolation
Ticket: GMT-015
Status: active
Topics:
    - analysis
    - minitrace
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/go-minitrace/cmds/query/run.go
      Note: Result envelope CLI contract
    - Path: repo://pkg/minitracedb/query.go
      Note: Non-null error arrays
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-17T15:13:24.590367-05:00
WhatFor: ""
WhenToUse: ""
---

# Persona review follow-ups: implementation design

## Goal and evidence

The first persona cohort completed 32 real Pi/Lunaroute runs against nine Claude, Codex and Pi transcripts. Its review identified one concrete type/runtime defect, repeated interface-discovery friction, generated analytical errors, and shortcomings in the evaluator's own isolation and answer contracts. This design turns those findings into focused changes while retaining Goja, SQLite, the Glazed CLI and the existing archive model.

This is a new implementation phase after the review was delivered. The frozen v1 prompts, answers, grades and collected sources remain historical evidence. They will not be edited to make the old experiment look better. New harness code and new runs use a separate v2 directory and evidence root. The native reader, adapter formats and provider replay behavior are unchanged unless a focused regression demonstrates a need.

## Decisions and scope

| Finding | Implementation | Acceptance evidence |
|---|---|---|
| Error envelopes contain null arrays despite array declarations | Initialize rows and columns on every query error path | Go and Goja negative-path tests |
| Permissive CLI stdout hides clipping diagnostics | Add --result-envelope to query run | Actual JSON output retains clipping and row-overflow metadata |
| execute(SQL-string) error is unhelpful | Reject with explicit alternatives | Exact argument-shape diagnostic test |
| Malformed normalized archive loses typed decode error | Recognize a narrow normalized shape before native fallback | Field/type error preserved; native inputs still recognized |
| Zero-tool sessions disappear from generated summaries | Add SessionToolCounts recipe rooted in sessions | Three-session fixture including zero; bound session filter |
| Native inspection shape is guessed | Full native TypeScript interfaces and executable projection | Fixture asserts nested coordinates, summaries and optional fields |
| Flags and command nesting are confusing | Command-scope table and runnable nested example | Embedded help smoke and executable examples |
| Prompt answers have ambiguous scopes | Explicit v2 JSON schema per task | Grader rejects wrong scope, duplicates and missing identities |
| Models write outputs into repository | Prebuilt hashed binary and OS read-only protection | Deliberate forbidden writes fail before model launch |
| Shell wrappers hide failed exit codes | Saved command wrapper with structured receipts | Nonzero subprocess exit retained and propagated |

The disproven RuntimeArchives helper hypothesis gets no product patch. Negative numeric metadata defaults remain an upstream scanner follow-up; valid-sibling isolation is already implemented and tested. No new parser, universal replay engine, broad SQL permission changes or compatibility adapter is needed.

## 1. Stable result envelopes

`pkg/minitracedb/query.go` owns the result contract. Every returned QueryResult must have non-null Columns and Rows, including nil-target, SQL validation, authorization, connection, execution and column-limit failures. A small constructor for error results makes the invariant explicit rather than relying on a custom JSON marshaler to disguise nil state. Strict-evidence rejection continues to clear partial rows while retaining diagnostics and columns.

The distinction between query() and queryResult() stays intentional: rows-only convenience methods throw on an error; envelope methods return the error string. An empty successful queryOne() returns null and its declaration must say so. execute() accepts a built recipe or an object with SQL and bound arguments; it will not silently reinterpret every argument shape.

```javascript
const result = db.queryResult('SELECT * FROM missing_table');
// result.rows === [] in shape, columns === [], complete === false,
// error contains the SQLite/authorization diagnostic.
if (result.error) throw new Error(result.error);
```

For execute('SELECT ...'), the error should explain: use queryResult(sql, ...args) for a SQL string, or execute({sql, args}) for a recipe. This preserves a narrow API while making recovery obvious.

Normalized input detection must be conservative. A parsed object with normalized identity/turn keys and a normalized companion field can retain its original decoding failure. A single native record with type/message/payload must still reach the native adapter. JSONL containing several objects is not a malformed normalized object merely because the first record looks familiar. Preserve the wrapped json.UnmarshalTypeError so callers can inspect it programmatically; include field and type context for humans.

## 2. CLI envelope mode

`query run --result-envelope` will emit one Glazed row containing the full QueryResult instead of emitting each selected SQL row separately. With `--output json`, the CLI's existing formatter therefore returns a one-element JSON array whose element is the envelope. Keeping the established formatter avoids a second output subsystem and keeps standard output/file settings usable. Documentation will show `.[0]` explicitly rather than implying a different top-level JSON shape.

```text
query target -> QueryResult
                    |
        +-----------+-------------+
        |                         |
  normal row mode          --result-envelope
  one output per SQL row   one output containing rows + metadata
```

The envelope includes columns, rows, count, complete, truncated, rowsTruncated and cellDiagnostics with stable JSON names. It is a reporting option, not permission to bypass strict evidence. Strict failures and query execution failures continue to return nonzero and emit no incomplete result rows; --run-record remains the machine-readable failure receipt. On successful permissive execution the envelope preserves every loss diagnostic. Existing row output remains the default because the user explicitly requested preserving that surface.

Tests will exercise UTF-8 clipping, row overflow, zero rows, ordinary row output and strict rejection with envelope requested. At least one executable CLI check will parse actual JSON, so an in-memory processor assertion cannot conceal formatting loss. Help will point directly to `go-minitrace help minitrace-schema` and explain where db.schema()/db.tables() fit.

## 3. Recipes and discoverable native types

SessionToolCounts is a small standard recipe, not a new aggregation framework:

```sql
SELECT s.session_id, COUNT(tc.tool_call_id) AS tool_count
FROM sessions s
LEFT JOIN tool_calls tc ON tc.session_id = s.session_id
WHERE (? = '' OR s.session_id = ?)
GROUP BY s.session_id
ORDER BY s.session_id;
```

The recipe uses actual tool rows rather than declared session metrics. COUNT(tc.tool_call_id) preserves zero; COUNT(*) would count the left-join placeholder and is wrong. The optional SessionID filter uses bound arguments, including strings that contain SQL punctuation. Tests compare the identity set and each count, not only the sum.

Native Inspect() declarations will match the Go result exactly: source coordinates, request status/diagnostics/references, shape/blob maps with raw JSON, compaction boundary/logical parent/metadata/summaries and optional neighboring messages. They must not advertise fields that the resolver consumes internally but does not return. A copyable projection example will reference the actual nested properties and be executed against a fixture. This is the specific remedy for the observed guessed-field null outputs.

## 4. Documentation and executable examples

Add one concise help page linking the full reference to the practical workflow. Its scope table will distinguish:

- query run flags: SQL selection, cell/row limits, strict evidence, cache policy, envelope and run receipt;
- query commands runtime flags: archive selection and repository paths, with database policies set by JS builders;
- conversion flags: common source selectors, with --run-record documented as supported by the Codex converter rather than inferred for every converter;
- JavaScript result styles: throwing rows-only methods versus inspectable envelopes.

The examples must show both a self-named single-verb file and a multi-verb file's full nested invocation. Include a bound-parameter query, the sessions-driven count recipe, and a native compaction projection. Test the examples at the public CLI/Goja boundary, then run the embedded-help smoke check. Do not scatter several nearly identical untested quickstarts through the repository.

## 5. A separate v2 testing harness

The old matrix is immutable. New code lives in ticket scripts/persona-testing-v2 and new logs under a separate private experiment root. The harness prepares a binary from the implementation checkout once, records its SHA256 and revision, and launches workers from their writable run directories. Workers never need to cd into the repository to execute the CLI. Corpus and fixture snapshots are selected explicitly and hashed before and after runs.

The first implementation targets the current macOS host through sandbox-exec. Its policy enforces no file writes under canonical repository, corpus, fixture and binary paths; it may additionally deny evaluator reads. Pi must still read credentials and maintain provider state outside those protected paths, so this is deliberately documented as enforced protected-path isolation rather than a complete hostile-code sandbox. Unsupported platforms fail closed before launching models. No prompt-only fallback is labeled sandboxed.

Before any model call, execute a probe inside the same policy: an ordinary write in the run directory succeeds, while writes/renames under each protected path fail. A policy startup error or a successful forbidden write aborts the experiment. Include symlink/canonical path coverage where the host supports it. A container/read-only mount backend can be added later with the same tests; do not add a container dependency merely for portability in this patch.

Each task has mandatory checks and optional exploration. Require an early answer.partial.json checkpoint with explicit completion state, then the final answer and debrief. The task schemas use synthetic_request_counts and real_request_counts separately, and archive_source_ordinals versus chronological_source_ordinals separately. Real totals explicitly exclude the fixture. Native paths may be absolute or unique raw-root-relative paths; duplicate aliases remain invalid. Every per-session result includes identities and zero-tool cases.

A command wrapper records argv, cwd, start/end, stdout/stderr paths and the true subprocess exit status, and returns that status to Pi. This avoids interpreting a shell's final echo or cat as the queried program's success. Direct shell work remains observable in the Pi trace; the wrapper cannot honestly claim that a worker used it for every command unless traces confirm that.

Fresh runs never overwrite old attempts. Run the relevant focused probes after each implementation milestone; after integration run a fresh small cohort before scaling to the full 32-slot matrix. Requested thinking settings, provider mapping, model usage and execution time remain separate observations. Correct field values, task completion, replay checks and workspace compliance are separate denominators.

## Validation and commit boundaries

1. Commit this design and the diary's start checkpoint before implementation.
2. Commit stable result/diagnostic changes after focused Go/Goja tests.
3. Commit the CLI envelope, recipe/native typing and executable help changes after integration tests and help smoke.
4. Commit the v2 harness after local sandbox/receipt/grader probes, then commit fresh-cohort receipts and conclusions separately.

At the integrated boundary run affected package tests, go build ./..., go vet ./... and the repository lint commands available on this machine. The known baseline macOS configuration-test failures and unavailable golangci-lint must remain explicit if still present; do not quietly reclassify them as passing. Product code and documentation stay frozen during each model cohort. Model feedback is reviewed afterward rather than patched into an active comparison.

The detailed diary records actual commands, failures, recovery decisions, commits and remaining uncertainty. This design is the implementation contract; the historical review remains the record of what the prior revision demonstrated.
