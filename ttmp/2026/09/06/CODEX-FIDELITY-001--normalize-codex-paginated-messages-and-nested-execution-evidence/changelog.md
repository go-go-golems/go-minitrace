# Changelog

## 2026-09-06

- Initial workspace created


## 2026-09-06

Created evidence-backed intern guide for paginated messages, structured executions, output blocks, unknown outcomes, and downstream query acceptance.


## 2026-09-06

After user upgraded BasicTeX, verified TeX Live 2026 and dependencies, rendered and visually reviewed eight-page guide, and successfully uploaded to /ai/2026/09/06/CODEX-FIDELITY-001. Receipt saved; implementation remains open.

## 2026-09-06

Validated three local Codex 0.153.3 paginated sources against checkout-built conversion: zero turns, missing execution fields, malformed outputs, and orphan links; saved baseline scripts and evidence with three legacy controls.

## 2026-09-06 — Implementation P1

Committed baseline and explicit implementation contracts in `443cfbd`. Printed overall plan and P1 start; added synthetic paginated/execution/output fixture and independent CLI oracle in `0f67556`. Existing Codex tests pass; eight desired fidelity assertions fail on the unchanged converter, recorded as the reproducible before-state. P1 completion receipt: `various/slips/02-p1-done.log`.

## 2026-09-06 — Implementation P2

Restored persisted message variants with native identity reconciliation and exact complementary mirror matching; removed speculative tool-to-message associations in `22b1f4e`. Full tests, lint, and Codex race tests pass. Six private sources have complete native-message line coverage and no orphan links; three paginated sources now yield 252/305/99 turns. Printed P2 start and completion; evidence and lint triage recorded in reference/03 and various/p2.

## 2026-09-06 — Implementation P3 milestones

Printed P3 start. Nullable outcomes now survive Go schema, SQLite, protobuf, TypeScript, and neutral UI rendering (`80d7bd5`); repaired existing frontend validation blockers in `ba54414`. SQL/API tests, Go-to-TypeScript roundtrip, 22 browser story tests, and inspected outcome screenshots are recorded in `a40ea81`.

Native CommandExecution lifecycle reconciliation and typed output decoding landed in `6e656e7`. The synthetic oracle passes all nine assertions. Independent private-source audit recovers 2,296 native executions and 155 failures with matching argv/output/outcomes and unchanged source hashes. All sources have zero malformed map outputs and zero orphan links. Full tests, lint, and race checks pass. P3 remains active for legacy/adversarial hardening; file evidence/counter consumers and final documentation/acceptance remain open.

## 2026-09-06

P3 hardening: reconcile legacy terminal outcomes after results, preserve terminal conflicts, and keep exec-stream completion-only outcomes unknown (9658081). Full tests, lint, and Codex race tests pass.


## 2026-09-06

Resumed with user-authorized additional 1M tokens. Hardened one-to-one native execution identity (f0d4811) and retained repeated/early output provenance and conflicts (4e61b7b). Full tests, lint and Codex race checks passed.


## 2026-09-06

P3 accepted against fresh private/synthetic audits; P3 completion and P4 start physically printed with receipts. P4 schema milestone f0a5377 adds typed targets, independent file outcomes, and SQLite v5 evidence projections; extraction and consumers remain in progress.


## 2026-09-06

P4 extraction milestone 24bea46: structural patch/redirect attempts and native FileChange effects; bounded shell grammar rejects unsafe inference. Independent audit matches 472 events and 995 targets; execution audit, full tests, lint and race checks pass. Counters/history/API integration remains active.

