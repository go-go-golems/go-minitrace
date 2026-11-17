---
Title: Review followup implementation and v2 validation results
Ticket: GMT-015
Status: active
Topics:
    - analysis
    - minitrace
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: ""
LastUpdated: 2026-11-17T18:18:09.462631-05:00
WhatFor: ""
WhenToUse: ""
---

# Review follow-up implementation and validation

This document describes the changes made after the original 32-run persona review. The historical review describes the previous revision; it remains evidence, not a live API manual. Design03 is the implementation plan, maintained help under `pkg/doc` describes the current API, and this document records what was delivered and how it was checked.

## Delivered product changes

The primary product checkpoint is acac40b. Queries still flow from Goja or the CLI into the same normalized SQLite engine. The changes make existing behavior easier to inspect and recover from rather than adding another execution layer.

- **Error results have stable arrays.** `pkg/minitracedb/query.go` constructs error envelopes with `rows: []` and `columns: []`. Go callers, JSON consumers and Goja now agree with the TypeScript declaration. An empty array is not proof of success: callers must inspect `error` and `complete`.
- **SQL-string misuse has actionable guidance.** `pkg/minitracejs/db_builder.go` tells an `execute(string)` caller to use `queryResult(sql, ...args)` or `execute({sql, args})`. Valid recipes retain their existing behavior.
- **Malformed archives retain their decoding error.** `pkg/minitracedb/convert.go` recognizes a conservative normalized shape after failed decoding, preserving the wrapped JSON field/type error. Recognized native records still reach their adapter. This is not a general JSON format detector.
- **CLI completeness is available on stdout.** `query run --result-envelope --output json` emits one Glazed row containing the result. Read `.[0].rows` only after checking `.[0].complete` and diagnostics. Empty successful queries still emit an envelope. Strict and SQL errors return nonzero before emission; `--run-record` remains the failure receipt.
- **Zero-tool sessions have a standard recipe.** `SessionToolCounts()` begins with sessions and LEFT JOINs tool calls. It counts the actual tool ID, not the left-join placeholder or stored metrics. Optional SessionID filtering uses bound parameters.
- **Native shapes are explicit.** TypeScript describes native coordinates, request statuses, shape/blob maps and nested compaction fields. `queryOne` is nullable. The native projection example is executed against a fixture covering absent neighbors and summary coordinates.
- **Practical help is executable.** `evidence-workflows` documents command-specific flags, byte limits, bound SQL and nested command paths. The checked-in multi-verb example runs in a CLI test. Single self-named verbs collapse their filename group; this exception is now explicit.

## How an intern should trace the implementation

Start at `cmd/go-minitrace/cmds/query/run.go`: it decodes flags, resolves input provenance, opens a query target and records the result. `command_runtime.go` is the emission boundary. Both output modes execute the query once and reject errors first; successful results either become individual rows or one envelope. The envelope keeps Go values intact, avoiding integer precision loss through a JSON/float64 round trip.

For JS, follow `db_builder.go` to the shared database runner. Recipe construction is in `query_view_session.go`; the recipe stores SQL and arguments rather than executing immediately. Type declarations live in `typescript.go`; native inspection implementation remains in `pkg/nativeevidence`. The public native projection example lives in `pkg/minitracejs/testdata/native_compaction_projection.js`.

```text
CLI settings / JS builder
          |
          v
normalized SQLite target --> QueryResult with completeness metadata
                                  |
                      reject errors / strict loss
                                  |
                         +--------+--------+
                         |                 |
                    ordinary rows     result envelope
```

A useful regression test asserts a semantic boundary: zero sessions survive an aggregate, bound punctuation cannot become SQL, clipped UTF-8 retains a diagnostic, or strict mode emits no rows. Avoid comparing object serialization order when the object comes from a Go map. The new recipe test initially made that mistake; the full suite exposed it and the assertion now projects fields explicitly while still checking row order.

## Validation and known limits

Focused Go/Goja checks pass for negative query paths, archive decode diagnostics, execute misuse, zero-tool counting, native projections, empty/clipped/overflow CLI envelopes and strict rejection. The corrected minitracejs suite passed three repetitions and the recipe test passed ten. Build, vet, embedded help and the pinned Glazed vet tool pass; the structured receipt is `sources/persona-v2-product-validation.json` and the reproducible driver is `scripts/16-validate-review-followups.py`.

The full Go suite still has two previously reproduced baseline macOS configuration-precedence failures: `TestNewCommandsCommand_LoadsConfiguredRepositoryFromAppConfig` and `TestNewCommandsCommand_XDGConfigOverridesHomeConfig`. `make lint` cannot invoke the absent `golangci-lint`; the separate pinned Glazed lint check succeeds. These are not reported as green gates. Signed negative numeric metadata defaults remain an upstream scanner limitation, and exact historical model replay remains outside this implementation.

## New experiment, separate evidence

The v2 harness is under `scripts/persona-testing-v2`. It builds one binary, records its hash/revision, copies the fixed nine-source corpus, and launches Pi from per-run scratch directories. A macOS policy denies writes under the repository, shared Git directory, copied/source corpus, fixtures, binary and control files. Direct, child-process and symlink write probes run before model invocation. Oracle reads are denied and an allowed scratch write is verified. This is protected-path enforcement, not a complete hostile-code sandbox; provider state and network outside protected paths remain available.

The command wrapper preserves real exit codes and stream hashes. An early partial answer is mandatory. Four task contracts separate synthetic and real data, source order and chronological order, and per-session identities. Host replay invokes generated commands with fixed argv rather than answer-provided shell text. Grading cannot prove every optional narrative or every claimed two-path comparison; inspect scripts and transcripts for those conclusions.

The first pilot found harness defects before the matrix: relative receipt stream paths needed canonicalization; task A needed exact count field names; all run directories needed creation before sibling read-denial policies were constructed; and valid JSON non-object answers needed a per-run error instead of a cohort crash. These fixes preserve the original pilot artifacts. D replay also checks session identity, not only the count. Six no-model harness regression tests pass, including actual OS denial probes.

## Fresh cohort results

Pilot and matrix completion receipts will be added after the runs finish. No result is inferred from a process merely starting or producing a partial answer. Requested model thinking levels are recorded independently of any claim about backend behavior.
