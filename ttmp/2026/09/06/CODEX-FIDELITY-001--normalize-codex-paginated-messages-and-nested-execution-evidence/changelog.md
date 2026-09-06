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
