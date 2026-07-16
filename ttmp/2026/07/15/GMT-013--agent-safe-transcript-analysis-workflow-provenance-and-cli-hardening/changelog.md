# Changelog

## 2026-07-15

- Created GMT-013 workspace for agent-safe transcript-analysis provenance and CLI hardening.
- Mapped current source discovery, adapter conversion, archive publication, manifest merge, normalized query, Glazed output, validation, and embedded-help behavior.
- Corrected the Codex collision diagnosis: distinct child headers are replaced by a later replayed parent `session_meta`, after which archive filename and manifest merge preserve the wrong parent identity.
- Documented evaluation failures: unsaved report SQL, directory-shape-sensitive manifest audit, pipeline-masked `find` errors, polymorphic Codex source metadata, and empty JSON for zero-row queries.
- Added an intern-oriented architecture and implementation guide with API sketches, receipt schemas, pseudocode, Mermaid flow, decision records, six implementation phases, test matrix, risks, open questions, and review order.
- Adopted a documentation-consolidation plan: no new feature help pages; merge `end-to-end-analysis` into `analysis-guide`, merge framework metadata mappings into adapter reference, and reduce redundant DuckDB stubs to one migration page.
- Added a chronological investigation diary and implementation task backlog.
- Related the design and diary to the key implementation and evaluation files.
- Validated the ticket with `docmgr doctor --ticket GMT-013 --stale-after 30`.
- Dry-ran and uploaded the bundled ticket documentation to reMarkable at `/ai/2026/07/15/GMT-013`.

## 2026-07-15

Completed evidence-backed design and diary for identity-safe conversion, collision-aware archives, reproducible query receipts, structured output, validation, and consolidated help architecture

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/ttmp/2026/07/15/GMT-013--agent-safe-transcript-analysis-workflow-provenance-and-cli-hardening/design-doc/01-agent-safe-transcript-analysis-architecture-and-implementation-guide.md — Primary design deliverable
- /home/manuel/code/wesen/go-go-golems/go-minitrace/ttmp/2026/07/15/GMT-013--agent-safe-transcript-analysis-workflow-provenance-and-cli-hardening/reference/01-investigation-diary.md — Chronological investigation record


## 2026-07-15

Expanded all GMT-013 implementation phases into stable, evidence-bearing tasks with dependencies, file scope, Done-when criteria, phase gates, and cross-phase review checkpoints

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/ttmp/2026/07/15/GMT-013--agent-safe-transcript-analysis-workflow-provenance-and-cli-hardening/reference/01-investigation-diary.md — Step 6 records the task decomposition rationale and review guidance
- /home/manuel/code/wesen/go-go-golems/go-minitrace/ttmp/2026/07/15/GMT-013--agent-safe-transcript-analysis-workflow-provenance-and-cli-hardening/tasks.md — Detailed phase-by-phase implementation plan


## 2026-07-15

P0.1-P0.4: recorded clean baseline, added redacted Codex child-to-parent replay fixture, captured the parent-ID overwrite, and locked session identity to the first native header

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/pkg/adapters/codex/convert.go — Prevents replayed parent metadata from replacing child archive identity
- /home/manuel/code/wesen/go-go-golems/go-minitrace/pkg/adapters/codex/convert_test.go — Regression test proving child ID and parent lineage


## 2026-07-15

P0.5-P0.12: completed green characterization tests for identity precedence, destructive archive overwrite, partial conversion publication, and zero-row JSON/empty-JS output; full Go suite passes

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/query/output_phase0_test.go — Captures empty streaming JSON behavior for Phase 3 replacement
- /home/manuel/code/wesen/go-go-golems/go-minitrace/pkg/minitrace/archive_test.go — Captures current overwrite behavior for Phase 1 replacement


## 2026-07-15

P1.1-P1.4/P1.6: added shared source identity and SHA-256 fingerprinting, Codex first-header inspection, and additive provenance evidence

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/pkg/adapters/codex/convert.go — Codex source inspection and child/parent provenance mapping
- /home/manuel/code/wesen/go-go-golems/go-minitrace/pkg/adapters/source_identity.go — Adapter-neutral fingerprint and source identity contract


## 2026-07-15

P1.7-P1.9: replaced silent archive overwrite with default collision rejection, fingerprint-idempotent reuse, explicit replacement, temporary-file publish, and Codex --collision flag

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/convert/codex.go — Explicit caller policy
- /home/manuel/code/wesen/go-go-golems/go-minitrace/pkg/minitrace/archive.go — Collision-safe archive publication


## 2026-07-15

P1.5/P1.10: added record-indexed Codex replay warnings and durable single-archive temporary-file publication with file/directory sync

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/pkg/adapters/codex/convert.go — Replay warning evidence
- /home/manuel/code/wesen/go-go-golems/go-minitrace/pkg/minitrace/archive.go — Durable individual archive publish


## 2026-07-15

P1 batch foundation: normalized, deduplicated, and sorted explicit source paths; added adapter-neutral conversion statuses/results for upcoming preflight and receipts

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/convert/sources.go — Deterministic source selection
- /home/manuel/code/wesen/go-go-golems/go-minitrace/pkg/adapters/conversion.go — Shared conversion outcome types


## 2026-07-15

P1.12: added strict Codex preflight that fingerprints all sources, detects conflicting native IDs, collapses byte-identical duplicates, and prevents output before later-source failure

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/convert/codex.go — Preflight before conversion/publication


## 2026-07-15

P1.11 progress: Codex conversion rows now expose preflight source fingerprint, identity basis, parent session ID, and warning count

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/convert/codex.go — Structured preflight provenance columns in conversion output


## 2026-07-15

P1.15 progress: Codex --run-record writes atomic conversion provenance JSON with sorted inspected inputs, outputs, collision policy, timestamps, and completion state

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/convert/codex.go — Codex run-record flag and successful-run integration
- /home/manuel/code/wesen/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/convert/run_record.go — Conversion receipt v1 serialization


## 2026-07-16

P1.13/P1 migration progress: staged collision-prechecked publication for Codex, Pi, and Claude Code, including in-memory Claude backlinks and incomplete Codex receipts (commits c991b97, 0fae467, 3242bd7, 1450ebc)

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/pkg/minitrace/batch.go — Staged publisher


## 2026-07-16

P2.1-P2.13: native archive/manifests/source/receipt validation, non-zero error findings, atomic manifest rebuild, docs, tests, and skill migration (commits cc0c234, b3017a9)

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/pkg/validate/archive.go — Phase 2 validation implementation


## 2026-07-16

P2 receipt reconciliation follow-up: failed Codex receipts now populate summary.failed consistently with failure entries (commit 189a240)

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/cmd/go-minitrace/cmds/convert/codex.go — Failure receipt summary reconciliation


## 2026-07-16

Pinned Go 1.26.5, verified go-minitrace tests/lint, verified Glazed formatter tests/lint and successful GoReleaser snapshot, and opened go-minitrace PR #25

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/go.mod — Go 1.26.5 and pinned Glazed formatter fix


## 2026-07-16

PR #25 review follow-up: restored Pi/Claude idempotent reruns with source fingerprints, bound query execution to resolved inventories, detected duplicate archive IDs, validated manifest file paths, and cleared GoSec G122/G302 findings (commit 0a867e7)

### Related Files

- /home/manuel/code/wesen/go-go-golems/go-minitrace/pkg/validate/archive.go — Review and GoSec fixes

