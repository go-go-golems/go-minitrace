---
Title: Implementation contracts and phased acceptance
Ticket: CODEX-FIDELITY-001
Status: active
Topics:
    - codex
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://cmd/go-minitrace/cmds/serve/outcome_test.go
      Note: Protobuf presence and badge tests
    - Path: repo://pkg/minitrace/outcome.go
      Note: Evidence-aware nullable outcome contract in 80d7bd5
    - Path: repo://pkg/minitrace/schema.go
      Note: Nullable outcomes and explicit record and file contracts
    - Path: repo://pkg/minitracecmd/core/history/file-history.js
      Note: Unsafe wrapper inference to replace
    - Path: repo://pkg/minitracedb/materialize.go
      Note: SQL outcome and file projections
    - Path: repo://pkg/minitracedb/outcome_test.go
      Note: SQL nullability and failure projection tests
    - Path: repo://proto/go_go_golems/minitrace/api/v1/sessions.proto
      Note: API outcome projection
ExternalSources: []
Summary: Concrete outcome, identity, counting, file evidence, and phase verification contracts.
LastUpdated: 2026-09-06T00:00:00Z
WhatFor: Make cross-consumer decisions explicit before implementation.
WhenToUse: Implementing or reviewing this ticket.
---



# Implementation contracts

This document resolves the design choices identified in the intern guide. The user authorized full ticket implementation. These are implementation decisions, not claims that the behavior already exists. Each decision must be covered by fresh tests and the final local-source acceptance audit.

## Phases and printed boundaries

| Phase | Deliverable | Required phase evidence |
|---|---|---|
| P1 | Explicit contracts and synthetic fixtures | Current failure reproduction; existing Codex package baseline; fixture inventory; plan/start/done print receipts |
| P2 | Message normalization and linkage | Mirrors collapse; repeated text retained; order/IDs stable; no orphan links; legacy/identity tests |
| P3 | Executions and typed outcomes | Lifecycle identity reconciliation; unknown/pending/cancelled preserved end-to-end; typed outputs decoded before truncation; schema/API/UI consumers tested |
| P4 | File evidence and analytical consumers | Multi-target attempts; no JavaScript false-branch or quoted-history writes; orchestration/execution counts separated; SQL/history tests |
| P5 | Complete acceptance and documentation | All tests and local CI; private before/after SQL; archive identities/receipts; UI outcome inspection; docs/skills/docmgr; final audit and printing |

Print an overall plan before implementation, then a start slip before work in each phase and a completion slip only after its evidence is inspected. Keep receipts in `various/slips/`. Commit focused milestones; record hashes in the implementation diary.

## Outcome contract

Change `ToolCallOutput.Success` to nullable `*bool`, consistent with the existing nullable session outcome and SQLite success column. Add an explicit outcome status with values `unknown`, `pending`, `succeeded`, `failed`, and `cancelled`. Preserve existing explicit true/false results from other adapters while updating their construction sites and consumers; do not add a legacy adapter or custom unmarshalling shim. Missing success decodes naturally as null. Bump the archive schema and SQLite cache/materialization versions when these changes land.

- `succeeded` means authoritative evidence of success for this record, with success=true.
- `failed` means authoritative failure for this record, with success=false.
- `unknown`, `pending`, and `cancelled` have success=null. Cancellation is not automatically process failure or success.
- Completed execution with numeric exit code is authoritative for the subprocess; a completed wrapper is not evidence that its children passed.
- Lifecycle completion without a usable outcome remains unknown. A start without completion stays pending.
- Conflicting authoritative outcomes must be diagnosed and retained as conflicting evidence; do not silently overwrite into a confident success.
- Recognized tool/transport result envelopes may establish an outcome for that tool record. Arbitrary JSON-looking stdout must not be interpreted as a transport envelope.

Project nullable success and outcome status through SQLite, serve API, generated protobuf/TypeScript, frontend types, badges, and tool rows. Unknown and pending must render neutrally rather than as a green check or red failure. Update presets and failure metrics to select explicit failures, not falsiness. Add tests for transport-versus-child outcomes. Existing known outcomes must remain known.

## Message contract

Normalize supported persisted representations before emitting final indexed turns. Record native message IDs, native turn ID, event kind, timestamp, and source line/ordinal. Prefer response messages when mirrored with completed-item messages; preserve canonical source content blocks without copying images/base64 or encrypted reasoning into metadata.

Deduplicate same native message IDs across representations. Different-ID mirrors may reconcile only across adjacent complementary message records with matching role/content and the same nonempty native turn ID. For legacy flat-text mirrors, compare the exact concatenation of native text blocks, not display-inserted separators; never normalize arbitrary whitespace. P2 covers legacy user/assistant mirrors as well as paginated user copies. Never merge repeated text solely by content, across turns, or across two records of the same representation. Preserve genuinely repeated messages, missing-ID messages, multi-block content, and fallback item-only messages. Conflicting same-ID content requires a diagnostic.

A native turn is not a normalized message index. Set `emitting_turn_index` only when explicit source linkage identifies an assistant message; do not infer a later final answer or nearest commentary as the emitter. Otherwise leave it null and retain native turn identity and an association diagnostic. Every non-null association must be reciprocal in `tool_calls_in_turn` and name an existing assistant turn. Legacy formats retain their supported content and tools, but remove invalid speculative associations rather than preserving a known bug.

## Execution and provenance contract

Persist wrappers and observed subprocesses as distinct records in `tool_calls`, each with an explicit record kind: ordinary tool call, orchestration, execution, or native file change. The additional `file_change` kind represents observed file effects, not a model invocation or executable child; count it separately. Namespace synthesized execution IDs using native item identity. A repeated lifecycle notification for the same native execution is one execution; identical commands with different execution IDs remain different executions. Associate response calls and execution records only via explicit native linkage. A shared turn ID is useful context but not parent proof. Retain uncertain parentage rather than guessing through event timing or JavaScript source.

Preserve command argv, native execution ID, cwd, lifecycle status and timestamps, source record references, and authoritative outcome. Decode `file://` cwd. For recognized shell `-c`/`-lc` argv expose the script as `command`; otherwise preserve argv and use a documented quoted display representation. Never run transcript code. Do not parse quoted message bodies into lifecycle events.

Reconcile start/completion and output-before-call, including duplicates, missing outcomes, cancellations, and conflicts. Known unsupported execution-bearing event shapes must yield fidelity diagnostics rather than a clean-looking archive. Keep diagnostics bounded, structured, and discoverable without copying full native payloads.

## Counting contract

Keep `tool_call_count` as the number of normalized tool records (the existing archive/list contract), and explicitly document that it is not a count of model-issued invocations. Add record-kind counts that separate observed executions and orchestration from ordinary tool calls. Operation counts must not count wrappers as executable children or infer writes from opaque orchestration source. Do not count file targets as extra tool calls. Project the counters/kinds to SQLite and relevant summaries so analysts can request model-call versus executed-process activity without double counting.

## Typed output contract

Decode supported strings and content arrays before output metadata parsing and truncation. Join readable text blocks with explicit boundaries; preserve block kinds and image/unsupported signals without base64 payload copies. Parse only recognized result contexts, keeping independently returned subprocess outcomes separate when a wrapper returns several results. Never choose one child's exit code as the wrapper's own. Preserve full-byte/hash/truncation metadata and a native source-line reference sufficient to recover full content from immutable source files.

## File evidence contract

Add an explicit list of structural file targets on tool input, with path, operation, evidence source, and attempted/confirmed semantics. Project all targets into the existing `files` table and expose evidence kind and nullable outcome. The scalar `file_path` remains a convenience, not a complete ledger. Count file touches separately from tool records.

- Direct patch Add/Update/Delete headers establish targeted attempts. Structured FileChange events with explicit native outcomes can establish observed file effects and must retain their native provenance. No text scan of opaque JavaScript or guardian/quoted histories may establish a write.
- Recognize literal shell redirects using a bounded syntax parser or deliberately narrow tested grammar; never match `>` by regex across arbitrary source. Heredoc bodies, quoted strings, process substitutions, dynamic names, and non-executed branches must not become confirmed writes. Unresolved/dynamic/unsupported forms produce diagnostics.
- Resolve relative targets against explicit execution cwd, not the final/session cwd. If no cwd is established retain the relative target as unresolved rather than fabricate an absolute path.
- A shell script's zero exit does not prove success of every statement. Syntactic targets are at most attempted; do not label attempts as confirmed. Paths in a search command are not reads.
- Update file-history to consume normalized structural evidence for Codex. Remove unsafe wrapper regex attribution, preserve existing supported non-Codex behavior, and document deliberately unsupported inference.

## Verification and completion gate

P1 baseline artifacts already demonstrate three local affected sources and three controls. Regenerate final private archives in a fresh directory; compare against independent native ID/event inventories and source fingerprints. Check complete source selection, unique identities, receipts, zero orphan associations, output decoding, failures and unknowns, execution and file counts, and provenance. The literal-tilde validator issue is a separate limitation and must not be passed off as a missing source or a semantic validation success.

Run `go test ./...`, relevant focused/race tests, `make all`, `make logcopter-check`, frontend type/tests/lint applicable to changed consumers, help-frontmatter smoke, and docmgr doctor. Inspect UI rendering for all outcome states using synthetic archives. Review generated code/assets and diffs; do not commit private native transcripts or accidental binaries/caches.

Before marking the durable goal complete, map every ticket acceptance invariant, implementation task, documentation update, commit/diary obligation, and all printed boundaries to concrete fresh evidence. Unverified requirements or unavailable tools/necessary decisions are blockers, not scope reductions.
