# Changelog

## 2026-04-08

- Initial workspace created

## 2026-04-09

- Rebuilt the comparison around a ticket-local `go-minitrace` archive instead of relying only on raw transcript/git inspection.
- Added numbered analysis scripts under `scripts/` for session conversion, boundary detection, timing, churn, failure inspection, and phase-1 tree comparison.
- Converted both source Pi sessions into `analysis/archive/` and synced four tool-call annotations that mark `phase-1-code-complete` and `phase-1-bookkeeping-complete` for both GPT-5.4 and MiniMax.
- Tightened the SQL methodology to compare the implementation window (`Add detailed tasks ...` → `phase-1-code-complete`) rather than GPT’s whole earlier research session.
- Rewrote the diary into a detailed retroactive investigation log.
- Wrote the main session analysis and the comparison findings docs focused on the end-of-phase-1 code state and the reasons MiniMax took longer.
- Bundled the findings + session analysis into a single PDF and uploaded it to reMarkable under `/ai/2026/04/09/MINIMAX-VS-GPT-COMPARE` as `MINIMAX vs GPT-5.4 Phase 1 Findings`.

## 2026-04-08

Uploaded findings bundle to reMarkable: /ai/2026/04/09/MINIMAX-VS-GPT/Cross-Model Transcript Analysis - Minimax vs GPT-5.4.pdf

### Related Files

- /ai/2026/04/09/MINIMAX-VS-GPT/Cross-Model Transcript Analysis - Minimax vs GPT-5.4 — Uploaded research bundle

