---
Title: Native JSONL evidence implementation notes
Ticket: GMT-015
Status: active
DocType: reference
Topics: [analysis, minitrace]
---

# Native evidence implementation

## Step 1: choose a separate opt-in surface

The prior Playback investigation required custom Python and Goja provider parsers because normalized archives omit native request/shape/blob records. Added `pkg/nativeevidence` and a small Goja builder in `pkg/minitracejs/native_evidence.go`. Existing normalized database/query builders remain unchanged. Root owns the `mt.native` module registration.

Current implemented API:

```javascript
const mt = require('minitrace');
const scan = mt.native().File('/path/to/native.jsonl')
  .MaxRecordBytes(16 * 1024 * 1024)
  .ForEach(record => {
    // record.source: {line, byteOffset, rawBytes}
    // record.value: parsed object, OR record.diagnostic
    // Return false to stop at this physical record boundary.
  });
// scan: sha256, hashScope, bytes, lines, malformed, stopReason

const evidence = mt.native().File('/path/to/native.jsonl')
  .MaxIndexBytes(64 * 1024 * 1024).Inspect();
// requests, blobs, shapes, compactions, diagnostics, scan, limitations
```

This is opt-in local filesystem reading, not a sandbox or arbitrary command executor. Each operation opens/closes its own source. Historical content is never evaluated as JavaScript. Inspect exposes native blob/shape contents intentionally; callers must project/redact them before publication.

## Step 2: streaming reader and exact byte identity

The reader consumes bounded chunks with bufio.ReadSlice. It hashes original consumed bytes, including CRLF, malformed JSON, invalid UTF-8 and oversized records. A too-large record is discarded while its remaining bytes are consumed and hashed; the next record remains readable. Invalid UTF-8 is diagnosed rather than silently normalized. Objects are required; scalars and null are malformed records.

Only reaching EOF produces `hashScope: whole_source`. An early callback return produces `consumed_prefix`, so its digest cannot be mistaken for whole-file identity. The callback runs synchronously and may throw; errors and cancellation propagate, with file close deferred on all exits. Cancellation is checked between reads; it does not interrupt an underlying filesystem read already blocked in the OS.

The streaming primitive retains at most one bounded raw record plus decoded data. Inspect deliberately builds an index, limited separately. Its limit accounts logical retained JSON bytes, not exact Go heap bytes. Exceeding it returns an error; JS throws rather than exporting a partial success. No private transcript fixtures are included.

## Step 3: request reconstruction and compaction inspection

The provider index retains shape/blob source coordinates and contents. Requests resolve against earlier records in the same file: known shape, integer nonnegative keep, available base prefix, supported string tail references and available blobs. Output status is complete or the first diagnostic category; the full diagnostics array can contain additional missingness. Categories include missing_shape, missing_base, missing_blob and unsupported. Duplicate identities retain the first object, report ambiguity, and prevent later referencing requests from being classified complete.

Requests are not retroactively completed by later blob records. Cross-file/fork recovery is not implemented. Base reference positions affected by unsupported data remain incomplete rather than silently certified. Wrapper parsing and instruction-byte attribution remain above this generic provider layer; no `instructions().Parse()` API is claimed.

Compactions retain explicit boundary coordinates, UUID/parent/logical-parent, native compactMetadata (including preserved graph metadata), preceding message, compact-summary coordinates, first following message and first following assistant. These physical coordinates help inspect a graph but are not automatic replay cuts. Result-arrival timing is not reconstructed; callers must not describe this as an exact causal checkpoint or hidden-memory revival.

## Step 4: synthetic validation and observed integration failure

The reader tests cover raw UTF-8/CRLF hashes, oversized records spanning multiple buffer chunks, malformed JSON, invalid UTF-8, final records without newline, early stop, callback failure and cancellation. Provider fixtures cover complete keep/tail chains, missing base/blob/shape, unsupported tail entries, future-only blobs, fractional/huge keep, duplicate identity and preserved compaction metadata whose logical parent appears physically later. Index-limit failure is asserted. Goja tests check lower-camel JSON fields, callback stop/errors, required path, invalid limits and malformed-record delivery.

`go test ./pkg/nativeevidence -count=1` passed. The initial combined package test could not compile because a concurrently authored provenance_test treated a rows-only slice as an envelope (`r.Error` / `r.Rows`); this was reported to the parent/owner and is unrelated to native evidence code. Integration tests will be retried after that change is corrected.

No Go database schema changes, private corpus ingestion, model calls or commits were performed by this subtask.

## Step 5: integration and numeric fidelity

After the concurrent test correction, both the native package tests and Goja binding tests passed. A restricted-cache attempt failed with a Go build-cache permission error; rerunning with the authorized normal cache access succeeded. Module registration was added by the parent.

A final evidence review added exact `rawJSON` to valid streamed records and retained shapes/blobs. The Go decoder uses json.Number so integer text beyond IEEE-754 safe precision is preserved; JavaScript's decoded `value` still follows ordinary JS number semantics, so callers needing exact numeric spelling should use `rawJSON`. Trailing second JSON values on one line are rejected. Synthetic tests include 9007199254740993 and CRLF preservation. Identical repeated blob/shape payloads deduplicate; conflicting payloads mark the identity ambiguous. No historical text is evaluated.

Remaining scope is deliberately explicit: no cross-file or future reference repair; no versioned instruction-wrapper parser; no certified checkpoint/replay or result-arrival timeline. These are follow-up features rather than claims hidden behind Inspect(). Index limits are logical accounting, not a measured heap ceiling. The JS bridge makes an additional JSON conversion for stable lower-camel fields, so consumers needing maximum throughput can use the Go streaming callback directly.
