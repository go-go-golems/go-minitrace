---
Title: Trustworthy transcript analysis architecture implementation and intern guide
Ticket: GMT-015
Status: active
Topics:
    - analysis
    - minitrace
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-17T14:01:36.471419-05:00
WhatFor: ""
WhenToUse: ""
---

# Trustworthy transcript analysis: architecture, implementation and intern guide

## 1. What this project does and why this change exists

`go-minitrace` converts coding-agent conversation logs into a common archive format, builds a normalized SQLite database from selected archives, and exposes analysis through SQL, an embedded JavaScript runtime, command-line commands and a web interface. A transcript records what a particular tool or harness chose to save. It is not automatically a complete recording of a model's input, a causal history of every action, or a reliable description of why the model made a decision.

This distinction became important during an investigation of Playback agent instructions. The investigation needed to count sessions and tools, identify instruction-file reads, reconstruct captured requests, measure instruction bodies, and inspect work around compaction boundaries. Python JSONL scripts established a baseline. JavaScript and SQL ports then exposed weaknesses in result clipping, native-record access, provenance and command discovery. This ticket implements focused improvements rather than replacing the existing architecture.

The most urgent defect was small enough to demonstrate with one expression. A three-byte cell limit applied to `abé` used to emit `ab�` without setting the result's truncation flag. A successful-looking query therefore changed its evidence. Increasing the default limit would merely postpone this failure. The implementation now preserves valid UTF-8 boundaries, reports loss explicitly, and offers a strict mode that withholds incomplete rows.

Other improvements are additive. We retain the archive schema and SQLite workflow, add original array positions to normalized tools, label token origins, introduce an opt-in native JSONL reader, and provide a small recipe execution bridge. Native inspection remains separate from ordinary normalized queries. No historical transcript is executed as JavaScript, and no model is invoked by these APIs.

This guide is written for an intern who needs to understand the repository, review the changes and continue the unfinished work. Examples marked as current API are implemented in this worktree. Pseudocode explains algorithms without promising an additional API. Testing statements describe verified scope; they do not imply that every repository package or every historical corpus has passed.

## 2. Start with the evidence model

There are three useful representations, each with a different purpose.

A **native source** is a provider or harness's original log, commonly JSONL. It can contain user and assistant messages, tool records, request references, cached content blobs, compaction metadata and operational events. Native formats differ between Claude Code, Codex, Pi and other adapters. Preserving original bytes matters when calculating identity: changing line endings, whitespace or decoding can change the source hash even if visible text looks similar.

A **normalized archive** is a `.minitrace.json` session document. The Go types live in `pkg/minitrace/`. Adapters under `pkg/adapters/` translate provider records into this model. Normalization makes common questions easy, but it can omit provider-specific information. A missing native request cannot be reconstructed from a normalized assistant message simply because both describe the same conversation.

A **query result** is a selected projection from a materialized SQLite database. It can be incomplete because of query limits even when the archive is complete. Conversely, an intact query result can faithfully return rows from an incomplete archive. The new `complete` field concerns query-result fidelity; it is not a universal evidence-quality certificate.

Keep the following questions separate during implementation and review:

- Did we read all bytes of the selected source, or stop early?
- Does a native request resolve all required references under this provider parser?
- Does normalization preserve the fields required by the question?
- Did the query return its selected rows and text without clipping or repair?
- Does the evidence establish the claim, or merely suggest a candidate for review?

For example, a recorded instruction body proves that text was present in that record. It does not prove the model used it. A retrospective model assessment is testimony, not access to the original model's hidden computation. None of the APIs in this ticket converts a relevance judgment into a causal fact.

## 3. Repository map and data flow

Read `AGENT.md` at the repository root before changing code. The relevant implementation areas are:

| Area | Responsibility | Starting files |
|---|---|---|
| `pkg/minitrace/` | Common session, turn, tool and provenance types | `builders.go`, session/type definitions |
| `pkg/adapters/` | Provider-specific conversion into archives | `claudecode/convert.go` and sibling adapters |
| `pkg/minitracedb/` | SQLite schema, materialization, query authorization, limits and corpus preflight | `schema.go`, `materialize.go`, `query.go`, `corpus_manifest.go` |
| `pkg/minitracejs/` | Goja-native builders and JavaScript-facing APIs | `module.go`, `db_builder.go`, `builders.go`, `query_view_session.go`, `native_evidence.go`, `typescript.go` |
| `pkg/nativeevidence/` | Opt-in native JSONL streaming and Claude request/compaction inspection | `reader.go`, `inspect.go` |
| `pkg/minitracecmd/` | Repository-backed SQL/JS command catalog | `catalog.go`, `repositories.go`, embedded `core/` |
| `cmd/go-minitrace/cmds/query/` | CLI flags, execution, help and query receipts | `run.go`, `run_record.go`, `commands.go`, `js_runtime.go` |
| `pkg/doc/` | Embedded help content | Existing query and JS documentation pages |
| `web/` | Browser frontend | Separate concern; this ticket does not redesign it |

The normal path remains:

```text
Provider log
    |
    v
Provider adapter --> normalized .minitrace.json archive
                             |
                             v
                   SQLite materialization/cache
                             |
                 +-----------+------------+
                 |                        |
                 v                        v
          direct SQL runner       Goja JS builders/recipes
                 |                        |
                 +-----------+------------+
                             v
                   result rows / envelope
                             |
                             v
                    CLI, report or UI
```

The new native path is deliberately adjacent:

```text
Original JSONL bytes
    |
    +--> ScanFile / mt.native().ForEach(...)
    |       bounded record + exact consumed-byte hash
    |
    +--> InspectFile / mt.native().Inspect()
            bounded logical index
            blobs + shapes + request references + compactions
```

This avoids forcing a large native request body into every normalized row or silently changing all existing importers. An analysis can use SQL for relational questions and native inspection for provider-specific evidence, joining the results by explicitly retained identity and source coordinates.

## 4. Opening a database and executing a query

The current JavaScript module is obtained with `require('minitrace')`. It is implemented in Go and executed by Goja. The environment supplies some Node-like modules, but it is not a promise of complete Node.js compatibility. Do not assume an API exists merely because a familiar module name can be required.

A repository-backed command can use the archive selection supplied through its runtime flags:

```javascript
const mt = require('minitrace');
const db = mt.db()
  .RuntimeArchives()
  .SQLiteMemory()
  .MaxRows(10000)
  .MaxCellBytes(16000)
  .StrictEvidence(true)
  .Timeout(30000)
  .Build();
try {
  const result = db.execute({
    sql: 'SELECT session_id, title FROM sessions WHERE session_id = ?',
    args: [sessionID]
  });
  if (result.error) throw new Error(result.error);
  return result.rows;
} finally {
  db.close();
}
```

The SQL parameter is bound through SQLite. It is not interpolated into a string. The database handle is closed in `finally`, including error paths. The query runner remains read-only and uses an allowlist/SQLite authorization boundary. Adding evidence features did not require granting arbitrary SQL writes or broadly enabling every SQLite function.

The existing recipe builder provides useful standard projections. A recipe contains SQL and its bound arguments, exposed through `sql()` and `args()`. `db.execute(recipe)` is a convenience bridge over that same mechanism:

```javascript
const recipe = mt.query().ToolRows().SessionID(sessionID).Build();
const result = db.execute(recipe);
if (result.error) throw new Error(result.error);
```

Plain `{sql, args}` objects are also supported. Missing or invalid recipe SQL produces a JavaScript type error. SQL execution problems use the result envelope, just like `queryResult()`. This distinction is intentional: the change does not rewrite every established error contract.

Use `db.query(sql, ...args)` when you deliberately want rows only. It already throws on the envelope's error, so enabling strict evidence also makes this simpler interface reject incomplete results. `queryOne()` follows the same rule. Use `queryResult()` or `execute()` when you need completeness diagnostics for review or logging.

## 5. Result fidelity and strict evidence

The implementation is in `pkg/minitracedb/query.go`; the JavaScript bindings are in `pkg/minitracejs/db_builder.go` and `builders.go`. The result envelope contains existing fields `columns`, `rows`, `count`, `truncated` and `error`, plus:

| Field | Meaning |
|---|---|
| `complete` | Always emitted. False on execution error, row overflow, cell clipping or invalid-UTF-8 repair. |
| `rowsTruncated` | The runner encountered another row after reaching the configured row budget. |
| `cellDiagnostics` | One diagnostic per lossy string/blob conversion, with output coordinates and byte lengths. |

A cell diagnostic includes zero-based `rowIndex` and `columnIndex`, `column`, `originalBytes`, `emittedBytes`, `truncated`, and optional `invalidUtf8`. The existing top-level `truncated` flag now includes cell loss as well as row overflow. Callers that specifically need row overflow should use `rowsTruncated`.

For the original failure example, permissive output is now:

```json
{
  "columns": ["text"],
  "rows": [{"text": "ab"}],
  "count": 1,
  "complete": false,
  "truncated": true,
  "cellDiagnostics": [{
    "rowIndex": 0,
    "columnIndex": 0,
    "column": "text",
    "originalBytes": 4,
    "emittedBytes": 2,
    "truncated": true
  }]
}
```

The three-byte budget cannot fit the final two-byte `é`, so the emitted prefix is two bytes. The implementation does not insert a replacement character solely because a limit falls inside a rune. If the source itself contains invalid UTF-8, the text is repaired for JSON presentation and explicitly marked incomplete. BLOB cells retain the existing text-oriented presentation; this is not a new byte-exact binary transport.

The preferred option is `MaxCellBytes`. `MaxCellChars` remains an explicitly supported deprecated alias with its historical byte semantics. In Go options, a positive `MaxCellBytes` takes precedence. In fluent JavaScript setters, the last explicit cell-limit setter wins. The limits facade supports `mt.limits().CellBytes(n).StrictEvidence(true).Build()`, passed to the database builder's `.Limits(...)` method.

Strict mode returns `error: "strict evidence: query result is incomplete"`, an empty row array and count zero. It preserves columns and diagnostics. The diagnostic coordinates refer to candidate output positions even though rows are withheld. It handles row overflow and cell loss consistently. An exact-limit result remains complete; simply returning exactly MaxRows rows does not establish overflow unless another row exists.

The algorithm is modest:

```text
execute authorized SELECT
for each returned row:
    if another row exceeds row budget:
        mark row overflow and incomplete
        stop
    for each cell:
        preserve non-text representation
        for text/blob:
            detect invalid original UTF-8
            repair invalid text for display, if necessary
            choose largest whole-rune prefix within byte budget
            record original/emitted sizes if information changed
if strict evidence and incomplete:
    keep diagnostics, withhold rows, return error envelope
```

Do not overinterpret `complete`. An author-written SQL `LIMIT 10` is part of the chosen query, not an accidental runner truncation. Missing native records, normalization omissions and duplicate SQL column labels are separate concerns. Use unique column aliases because the established row representation is a map keyed by column name.

## 6. Ordering, status and token provenance

`tool_calls.source_ordinal` records a tool's zero-based position in the normalized archive array. It is scoped to the session. It is not a native line number, an execution clock or a tool-result arrival coordinate. The existing ToolRows chronological ordering is unchanged; consumers needing archive order can explicitly request it:

```sql
SELECT session_id, source_ordinal, tool_call_id, arguments_json,
       success, output_status, outcome_status
FROM tool_calls
ORDER BY session_id, source_ordinal;
```

`arguments_json` exposes the serialized arguments directly. `output_status` is the archive's explicit status. `outcome_status` is the derived interpretation, which may use success or other evidence. Nullable success remains nullable. An absent status must not become an invented success. The existing Go archive model represents absent and empty status with the same empty string, so those two original encodings are still indistinguishable.

The schema is now version 6. Schema-version input to cache keys invalidates earlier version 5 materializations. This is a cache rebuild policy, not a migration system for arbitrary SQLite files supplied by users. Read `schema.go`, `materialize.go` and the new provenance tests together before extending the projection.

Token usage requires similar discipline. A provider-reported input field, a missing field and an estimate of tool-output size are different measurements. Token recipes now include `token_origin`, `token_unit`, `token_method` and `reported_usage_fields`. Existing numeric zero filling remains, but the labels distinguish unknown accounting from explicitly recorded zero.

Tool estimates use `estimated_tokens` and `round_utf8_bytes_div_4`. The fallback text measurement now uses SQLite BLOB byte length, aligning it with `full_bytes`; this corrects the former mixed character/byte arithmetic for non-ASCII output. It remains a heuristic rather than tokenization or billing. `reported_usage_fields` counts non-null usage fields, not requests or complete usage records.

These labels make mistakes detectable, but they do not prevent every caller from summing incompatible numbers. Do not combine estimated output tokens with reported prompt usage merely because both occupy a `total_tokens` column. Transcript views can also repeat accounting in message/reasoning presentations. Treat those as renderings, not necessarily disjoint additive observations.

## 7. Corpus identity before ingestion

A collection of archive paths is not yet a defensible study corpus. Overlapping globs can select the same file twice; recovered and live directories can contain equal copies; different byte contents can claim the same normalized session ID. Previously, path deduplication alone did not resolve those cases.

The current additive preflight is `minitracedb.BuildCorpusManifest(paths)` in Go and `mt.corpusManifest(paths)` in JavaScript:

```javascript
const manifest = mt.corpusManifest(selectedArchivePaths);
if (manifest.conflictCount > 0) {
  throw new Error('Choose explicitly between conflicting session variants');
}
const uniquePaths = manifest.entries
  .filter(entry => entry.duplicateOf === null)
  .map(entry => entry.path);
```

The manifest reports absolute paths, normalized and original IDs, source format/fingerprint, SHA-256, byte size, duplicate indices and conflict flags. `duplicateOf` is a zero-based entry index, so testing it for truthiness would incorrectly treat duplicate index zero as absent. Equal bytes can be identified; multiple hashes for one normalized ID are conservatively conflicts, even if the difference is whitespace or metadata.

This helper does not select a winner, modify ingestion, strip child prefixes or infer semantic equivalence. Legacy loading behavior remains unchanged unless the caller explicitly uses the manifest. Resolve conflicts using study-specific provenance, record the decision, then pass the selected paths into the existing database builder. A filename is not a substitute for identity stored in the document.

## 8. Native JSONL streaming without a whole-file JavaScript string

The streaming reader is `pkg/nativeevidence/reader.go`. It exposes `ScanFile` for a path and `Scan` for an `io.Reader`. The Goja surface is implemented in `pkg/minitracejs/native_evidence.go`:

```javascript
const mt = require('minitrace');
const counts = {};
const scan = mt.native().File('/path/to/native.jsonl')
  .MaxRecordBytes(16 * 1024 * 1024)
  .ForEach(record => {
    if (record.diagnostic) return;
    const kind = record.value.type || 'unknown';
    counts[kind] = (counts[kind] || 0) + 1;
    // Return false only when intentionally accepting a prefix scan.
  });
if (scan.hashScope !== 'whole_source') {
  throw new Error('This analysis requires whole-source identity');
}
```

Each record has `source.line`, `source.byteOffset` and `source.rawBytes`, plus a parsed object or diagnostic. Valid records also retain `rawJSON`. Lines are one-based physical coordinates; offsets identify bytes, not conversation positions. Malformed JSON, invalid UTF-8, scalar/null records and oversized records are diagnosed rather than silently accepted as ordinary objects.

The scanner hashes the original consumed bytes, including line endings and malformed records. A record exceeding MaxRecordBytes is discarded while the remainder is consumed and hashed, allowing the next line to be processed. The streaming primitive retains a bounded record plus its decoded representation, not a whole decoded source string and a second split-lines array.

Only reaching EOF yields `hashScope: "whole_source"`. Returning false from the callback stops at the current physical record boundary and reports `consumed_prefix`. An early digest is useful but must not be published as the whole-file hash. Files close on normal completion, callback errors, cancellation and early termination. Cancellation is checked between reads; it cannot promise to interrupt an OS filesystem read already blocked.

The Go decoder preserves number spellings using `json.Number`. JavaScript's decoded `value` follows normal JS numeric precision, so a value above the safe-integer range can be rounded there. Use `rawJSON` when exact numeric spelling matters. Do not parse it back into ordinary JavaScript numbers and assume precision has been restored.

The module now also exposes `mt.utf8ByteLength(text)`. It provides the narrow byte-length primitive needed by analysis without allocating a temporary Buffer. The external `buffer` module's `Buffer.byteLength` capability is unchanged. This is an explicit minitrace helper, not a claim of full Node compatibility.

## 9. Native request inspection and honest incompleteness

`mt.native().Inspect()` builds a provider-oriented index on top of the scanner:

```javascript
const evidence = mt.native().File('/path/to/native.jsonl')
  .MaxRecordBytes(16 * 1024 * 1024)
  .MaxIndexBytes(64 * 1024 * 1024)
  .Inspect();
const incomplete = evidence.requests.filter(r => r.status !== 'complete');
```

The result contains scan statistics, request records, blob and shape maps, compactions, diagnostics, retained logical index bytes and limitations. It intentionally exposes native payload contents. Before publishing a report, project only the needed fields and redact sensitive material; do not dump an entire inspection object merely to display request counts.

The current resolver understands Claude-style shape/blob/request records. A request can retain a prefix of an earlier request's references with `keep`, then append `tail` references. The provider layer checks shape availability, a nonnegative integer keep value, the earlier base and its available prefix, supported reference forms, and referenced blobs. The simplified algorithm is:

```text
when a shape/blob arrives:
    record identity, raw JSON, payload and source coordinates
    diagnose conflicting reuse of the identity
when a request arrives:
    validate keep and tail
    resolve its base only from earlier records in this file
    combine retained base references and new tail references
    check the referenced shape and every required blob
    retain all diagnostics
    classify complete only when the required evidence is available
```

Here, request `status: "complete"` means the supported reference graph resolved locally. It does **not** validate the provider's entire request schema, certify every payload field, or prove that the provider executed the request. Shape/blob identity presence is a narrower guarantee than semantic payload validity.

Missingness is explicit: categories include `missing_shape`, `missing_base`, `missing_blob` and `unsupported`. The first diagnostic category supplies request status, while the complete diagnostic list can carry several problems. Conflicting identities retain the first object but mark ambiguity so later dependent requests are not called complete. Identical repeated shape/blob payloads deduplicate.

The resolver does not retroactively complete a request when a missing blob appears later. It does not search other files, follow fork stores or recover garbage-collected bytes. These are deliberate boundaries, not implementation accidents hidden by a boolean. A partial request is not evidence of zero instructions or zero prompt tokens.

MaxIndexBytes is separate from the streaming record limit. Inspection retains data and therefore cannot have the same memory profile as a counting callback. The budget measures logical retained JSON bytes, not exact Go heap usage. Exceeding the budget throws an error rather than exporting partial success. A large-corpus heap benchmark remains useful future work.

Instruction-wrapper parsing is not implemented here. There is no current `instructions().Parse()` API, no versioned Claude memory grammar and no automatic instruction-body-byte attribution. The provider index gives future extractors trustworthy references and coordinates. A wrapper-specific extractor can be added later without mixing a fragile Markdown grammar into SQLite or pretending every provider has the same prompt layout.

## 10. Compaction inspection is not model replay

Compaction records describe a conversation transformation. Their physical position in JSONL can differ from the graph order of preserved messages. A logical parent may appear later in the source. A file reattachment may resemble a Read result without recording a new read action. The Playback investigation found that revived interviews could mislabel preserved pre-boundary actions as work occurring after a summary.

The current native inspection exposes boundary coordinates, UUID, parent UUID, logical parent, compactMetadata, preceding message, compact-summary coordinates, first following message and first following assistant. These are inspection aids. They do not certify a replay cut or reconstruct exactly what an old model saw.

```text
physical source order:
  old actions -> boundary -> attachments -> summary -> preserved records

possible conversation relationships:
  boundary.logicalParent ---------> preserved older message
  summary describes actions that occurred before the boundary
```

M09 remains open: normalized tool emission coordinates do not establish when a result became available. A packet capped at turn100 can still contain the eventual result of a tool emitted at turn99. Without result-arrival provenance, it is a retrospective packet, not a certified causal snapshot.

Do not add an automatic model-resume method to disguise this limitation. Temporary session registration, identity rewriting, model calls, budgets, cleanup and interview adjudication belong in external orchestration. This ticket provides data and diagnostics, not permission to run historical instructions. A future export helper should first prove graph membership and arrival boundaries with synthetic fixtures.

## 11. CLI behavior, catalog recovery and dependency issues

The direct SQL CLI now exposes `--max-cell-bytes`, `--strict-evidence`, `--cache-mode` and `--cache-dir`. The legacy `--max-cell-chars` remains documented as bytes. Cache modes include `auto`, `memory`, `disk` and `none`; memory/none avoid disk-cache writes. A read-only SELECT can otherwise legitimately require a cache write while materializing archives, which previously surprised restricted environments.

```bash
go run ./cmd/go-minitrace query run \
  --archive-glob '/path/to/selected/*.minitrace.json' \
  --sql 'SELECT session_id, title FROM sessions ORDER BY session_id' \
  --max-cell-bytes 16000 --strict-evidence \
  --cache-mode none --run-record /tmp/query-receipt.json --output json
```

Query receipts now retain completeness/cell diagnostics and relevant options. Existing `--strict` reproducibility behavior and `--allow-truncated` need to be understood separately from an explicit `--strict-evidence` request. Read `run.go` and `run_record.go` together rather than treating the flags as synonyms.

Repository-backed command loading has an additive diagnostic mode. `LoadCatalogWithDiagnostics` isolates malformed source files so valid siblings remain usable. The CLI reports skipped files. Strict `LoadCatalog` remains available, and structural collisions or broken aliases still fail closed. A broken higher-priority override must not silently fall through to an older lower-priority source.

The negative-literal parser problem is only partly addressed. A metadata default such as `-1` is still rejected by the pinned go-go-goja scanner. Catalog isolation limits the blast radius but does not add unary-expression support. Ordinary command help now includes custom sections, improving discovery of required fields. These are distinct fixes and should be tested separately.

Three existing repository issues provide broader context:

- [Issue20](https://github.com/go-go-golems/go-minitrace/issues/20) is closed: native module self-registration already exists. Do not build a second registration framework to solve a completed issue.
- [Issue23](https://github.com/go-go-golems/go-minitrace/issues/23) remains open for JS-first onboarding and documentation consolidation. This guide and type declarations contribute, but they do not constitute a complete redesign of public onboarding.
- [Issue24](https://github.com/go-go-golems/go-minitrace/issues/24) remains open for root-flag behavior, array-return wrapping, Pi summary promotion and deprecated DuckDB doctor behavior. These are not all solved by query fidelity or catalog diagnostics; keep their acceptance cases distinct.

## 12. Disposition of the original M01–M14 findings

The labels below refer to the Playback port audit, not new issue numbers. “Implemented” describes work in this branch, not a released version or an assertion that every original aspiration has been met.

| Finding | Current disposition | Remaining qualification |
|---|---|---|
| M01: silent cell loss | Implemented: UTF-8-safe clipping, cell diagnostics, completeness and strict rejection | Complete concerns query output, not source truth. |
| M02: misleading cell units | Implemented: preferred byte API plus explicit legacy alias | No general code-point-limit API; bytes are the declared contract. |
| M03: native request evidence absent | Partially implemented: separate opt-in native index and file-local resolver | Not added to every adapter/archive; no cross-file recovery or instruction-wrapper parser. |
| M04: mixed token origins | Implemented additive labels and consistent UTF-8-byte estimate | Arbitrary consumer aggregation is not automatically prohibited. |
| M05: no streaming native reader | Implemented callback scanner, coordinates, diagnostics and hash scope | Inspection builds an index; logical limits are not measured heap ceilings. |
| M06: signed defaults/catalog failure | Partially implemented: invalid-file isolation and diagnostics | Signed numeric metadata AST support remains in the upstream scanner. |
| M07: hidden required help fields | Implemented custom sections in ordinary CLI help | Full JS onboarding consolidation remains issue23. |
| M08: archive order inaccessible | Implemented source_ordinal and direct projections | Recipe chronological order stays unchanged; explicit ordering required. |
| M09: result-arrival/replay timing | Deferred | No certified causal packet or replay export. |
| M10: corpus identity | Implemented opt-in manifest | No automatic conflict resolution or enforcement during legacy ingestion. |
| M11: disk cache surprises | Implemented explicit CLI cache selection and directory | Callers must choose a policy; auto may still write disk. |
| M12: missing byte-length primitive | Implemented `mt.utf8ByteLength` alternative | External Buffer.byteLength and complete Node compatibility are unchanged. |
| M13: inconsistent error shapes | Clarified and tested existing envelopes/throws | No global breaking conversion of every error to exceptions. |
| M14: recipe bridging/projections | Implemented execute(recipe), arguments_json and output_status | Original/derived status remain separate; no richer arbitrary interpreter. |

The pragmatic decision is to keep the useful database and builder architecture, fix evidence loss first, then add narrowly scoped provenance and native access. Replacing everything with a generic dependency graph, a custom query interpreter or a universal replay engine would increase the surface area before the underlying contracts were reliable.

## 13. Validation, limitations and how to review the patch

Focused database, JavaScript, native-evidence and catalog packages have been exercised, including synthetic regressions. The fidelity-specific command under `scripts/query-fidelity/fidelity-probe.js` also passed through the actual CLI and Goja runtime. It uses synthetic content rather than private transcripts. The original three-byte probe now emits `ab` with 4-to-2 byte diagnostics; strict mode emits no rows.

The focused tests cover:

- Exact and overflowing ASCII, accented and supplementary UTF-8 boundaries; invalid original UTF-8; text BLOBs; oversized JSON; combined row/cell overflow; strict rows-only and envelope interfaces.
- Database and limits-builder aliases, bound recipe parameters, Goja JSON field casing, nullable values and invalid recipe shapes.
- Source ordinal despite conflicting ID order, original versus derived status, known-zero versus missing usage, token origin labels and schema cache invalidation.
- Equal corpus copies, conflicting same-ID variants, missing identity and duplicate index zero.
- CRLF/raw-byte hashes, early stop, malformed/oversized records, callback failure, cancellation, exact raw numeric spelling and invalid UTF-8.
- Request keep/tail chains, missing references, future-only blobs, unsupported forms, duplicate identities and preserved compaction metadata.

Implementation commit `60ae64a` passed the four focused packages and three new CLI tests. `go build ./...`, `go vet ./...`, and the pinned Glazed lint check passed. The full `go test ./...` run had two CLI configuration-test failures; both also reproduced on baseline revision `19d18b9`. They are recorded baseline failures, not a green full suite and not evidence that this patch fixed configuration handling. `make lint` stopped because `golangci-lint` was not installed. Logs are retained in ticket `sources/validation-all-tests.txt`, `validation-baseline.txt`, `validation-vet.txt`, `validation-glazed-lint.txt`, and `validation-lint.txt`. The new native API has not been rerun for parity against the private historical corpus. No broad historical replay, multi-gigabyte heap guarantee, cross-provider native-resolution parity or browser UI regression suite is claimed here.

Useful commands from the worktree root are:

```bash
go test ./pkg/minitracedb ./pkg/minitracejs ./pkg/nativeevidence ./pkg/minitracecmd
go test ./cmd/go-minitrace/cmds/query
go run ./cmd/go-minitrace query commands \
  --query-repository ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide/scripts/query-fidelity \
  fidelity-probe --output json
go run ./cmd/go-minitrace help --all
```

Start review at `query.go` and its dedicated fidelity tests, then follow the JS bridge. Next inspect schema/materializer changes alongside provenance tests. Finally inspect the native reader separately from the provider index. This order makes each guarantee understandable before reviewing the composition. Public TypeScript declarations live in `pkg/minitracejs/typescript.go`; they should be checked against executable runtime behavior, not treated as independent truth.

## 14. Follow-up work for an intern

Begin with small, independently reviewable changes. Each task should state the current failure, the desired contract, a synthetic fixture and a verification command. Do not begin by importing a large private transcript archive into unit tests.

**First: reconcile documentation and remaining CLI configuration failures.** The two configuration failures reproduce on baseline; isolate their environment dependency in a separate fix. Preserve that baseline evidence instead of labeling them fixed or blaming the new code. Check normal help, long help and required parameters with a small repository-backed command. Add executable examples to onboarding so issue23 moves toward closure without duplicating another stale API reference.

**Second: signed numeric metadata.** Work in the actual go-go-goja scanner dependency. Recognize safe unary plus/minus on numeric literals without evaluating arbitrary JavaScript. Test positive, negative, zero, unsupported expressions and a valid sibling next to an invalid file. Update dependency pinning deliberately; catalog isolation alone is not the parser fix.

**Third: stricter provenance composition.** Add helpers that refuse summing incompatible token units, or explicitly group them in reports. Test unknown accounting separately from a reported zero. Consider an optional corpus-ingestion policy that consumes a manifest and requires declared conflict handling; do not silently change legacy winner selection while adding it.

**Fourth: native extraction extensions.** Build a versioned instruction-wrapper extractor above the generic native index. Fixtures should cover missing wrappers, multiple sections, malformed delimiters, duplicate bodies and exact byte attribution. Only then compare against the frozen Python/native-port receipts. A wrapper parser must report unknown values rather than turning failed reconstruction into zero bytes.

**Fifth: causal snapshots.** Before writing a checkpoint export API, preserve tool-result arrival coordinates and test a tool emitted before a cutoff whose result arrives afterward. Add preserved-segment and missing-parent graph fixtures. State which provider formats can support the contract. A safe “unsupported” result is better than a plausible but false replay.

**Sixth: performance qualification.** Benchmark a multi-gigabyte synthetic stream, maximum-size records and an index near its logical budget. Measure peak memory and cancellation latency. Document the distinction between a streaming callback and an indexed inspection. Optimize only after measurements identify a material bottleneck; avoid introducing another cache merely because a parser exists.

Keep model execution and infrastructure orchestration outside the core library. The APIs should help an agent inspect evidence through ordinary JavaScript, not quietly invoke external agents, mutate transcript stores or embed a new workflow engine. A successful migration removes repeated defensive parsing while preserving clear ownership of source identity, projection limits and analytical interpretation.

## 15. Reference trail

- [Query fidelity implementation notes](../reference/02-query-fidelity-notes.md): exact API contract, failures and CLI probe.
- [Archive/token/corpus provenance notes](../reference/03-provenance-notes.md): source ordinal, status, token labels and manifest semantics.
- [Native evidence notes](../reference/04-native-evidence-notes.md): streaming, request completeness, raw numbers and compaction limits.
- [Implementation diary](../reference/01-implementation-diary.md): chronological integration work and final validation status.
- `pkg/minitracedb/query_fidelity_test.go` and `pkg/minitracejs/query_fidelity_test.go`: focused result-fidelity regressions.
- `pkg/nativeevidence/native_test.go`: reader and provider-index fixtures.
- `pkg/minitracedb/corpus_manifest.go` and provenance tests: explicit archive identity preflight.
- `cmd/go-minitrace/cmds/query/run.go` and `run_record.go`: actual CLI options and receipts.

The source implementation and executable tests take precedence over any illustrative pseudocode. Update this guide when the remaining capabilities land, keeping historical limitations in the diary rather than silently rewriting what this branch originally proved.
