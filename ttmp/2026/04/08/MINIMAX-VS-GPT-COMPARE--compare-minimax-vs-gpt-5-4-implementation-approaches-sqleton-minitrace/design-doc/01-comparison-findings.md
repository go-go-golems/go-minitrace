---
Title: Comparison Findings
Ticket: MINIMAX-VS-GPT-COMPARE
Status: active
Topics:
    - code-review
    - analysis
    - minimax
    - gpt
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../../../tmp/gpt-phase1-2wA3Rx/pkg/minitracecmd/assets.go:GPT phase-1 embedded asset loader, absent from MiniMax phase 1
    - Path: ../../../../../../../../../../../../tmp/gpt-phase1-2wA3Rx/pkg/minitracecmd/catalog.go:GPT phase-1 catalog loader snapshot
    - Path: ../../../../../../../../../../../../tmp/gpt-phase1-2wA3Rx/pkg/minitracecmd/compiler.go:GPT phase-1 compiler snapshot
    - Path: ../../../../../../../../sqleton-minitrace-minimax/go-minitrace/pkg/minitracecmd/catalog.go
      Note: MiniMax phase-1 catalog implementation under review
    - Path: ../../../../../../../../sqleton-minitrace-minimax/go-minitrace/pkg/minitracecmd/catalog.go:MiniMax phase-1 catalog loader
    - Path: ../../../../../../../../sqleton-minitrace-minimax/go-minitrace/pkg/minitracecmd/compiler.go
      Note: MiniMax phase-1 compiler implementation under review
    - Path: ../../../../../../../../sqleton-minitrace-minimax/go-minitrace/pkg/minitracecmd/compiler.go:MiniMax phase-1 compiler
    - Path: ../../../../../../../../sqleton-minitrace-minimax/go-minitrace/pkg/minitracecmd/parse_alias.go
      Note: MiniMax phase-1 defensive alias parsing improvement
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/scripts/12-phase1-timing.sql
      Note: Timing evidence for why MiniMax took longer
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/scripts/12-phase1-timing.sql:Timing evidence for the implementation-window comparison
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/scripts/13-bash-failures-preboundary.sql
      Note: Failure snippets from the implementation window
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/scripts/14-compare-phase1-tree.sh
      Note: Phase-1 code tree comparison
ExternalSources: []
Summary: Prioritized findings about the end-of-phase-1 code state and the causes of MiniMax’s slower path to phase-1 completion.
LastUpdated: 2026-04-09T01:08:00Z
WhatFor: Turn the session-analysis evidence into concrete review findings and recommendations for MiniMax 2.7.
WhenToUse: Read this after the analysis doc if you want the concise review conclusions and actionable recommendations.
---


# Comparison Findings

## Executive Summary

The end-of-phase-1 comparison does **not** show that MiniMax produced bad code. It shows something subtler:

1. **MiniMax’s resulting phase-1 package is narrower than GPT-5.4’s phase-1 endpoint**.
2. **MiniMax spent noticeably longer getting there because it had to repair several local implementation/test/lint issues in sequence**.
3. **MiniMax’s code quality is mixed but respectable**: it loses on package completeness and time-to-finish, but wins in some test depth and defensive parsing details.

So the main lesson for MiniMax 2.7 is not “write fundamentally different code.” It is:

> **Keep the same general design, but drive it in smaller validated slices and checkpoint package completeness earlier.**

## Prioritized findings

| # | Severity | Finding | Evidence anchor |
|---|---|---|---|
| 1 | Significant | MiniMax phase 1 ends with a less complete package boundary than GPT phase 1 | GPT phase-1 tree has `assets.go`, `assets_test.go`, `types_test.go`, `core/timing-analysis.sql`; MiniMax does not |
| 2 | Significant | MiniMax lost time to local repair loops, not to broad research overhead | 19.0 min vs 11.8 min to code complete; MiniMax has denser test/lint/build churn and explicit failure snippets |
| 3 | Minor-positive | MiniMax’s final phase-1 code is not low quality overall; some files are more heavily tested and more defensive than GPT’s | Larger test files, richer `framework-summary.sql`, explicit alias `query:` rejection |
| 4 | Recommendation | MiniMax 2.7 should adopt smaller slice boundaries and explicit boundary annotations/checkpoints during implementation | GPT’s small-commit progression vs MiniMax’s big-burst stabilization loop |

## Finding 1: MiniMax phase 1 ends with a less complete package boundary than GPT phase 1

### What

At the annotated phase-1 boundary, GPT’s package already includes:

- `pkg/minitracecmd/assets.go`
- `pkg/minitracecmd/assets_test.go`
- `pkg/minitracecmd/types_test.go`
- `pkg/minitracecmd/core/timing-analysis.sql`

MiniMax’s phase-1 package does not include those files.

### Why it matters

This means GPT phase 1 ends with a package that is not only parser/compiler/catalog-capable, but also already has:

- an embedded-source-root helper,
- a smoke test for the embedded catalog path,
- a small foundational test for types/source-kind,
- and one more built-in command artifact.

MiniMax’s phase-1 endpoint is therefore **narrower in scope**, even though the files it does have can be richer and more heavily tested.

### Review interpretation

This should be described as a **package-completeness gap**, not as a total code-quality failure. The MiniMax package is still valid and testable; it just ends sooner than GPT’s phase-1 endpoint.

### Fix

For MiniMax 2.7, add an explicit “phase-1 completeness checklist” that includes:

1. parser/compiler/catalog code,
2. foundational tests,
3. embedded asset loader + smoke test,
4. the intended built-in command set for that phase.

### Evidence

- GPT phase-1 tree: `scripts/14-compare-phase1-tree.sh`
- GPT phase-1 assets helper: `/tmp/gpt-phase1-2wA3Rx/pkg/minitracecmd/assets.go`
- MiniMax phase-1 package dir: `/home/manuel/workspaces/2026-04-08/sqleton-minitrace-minimax/go-minitrace/pkg/minitracecmd`

## Finding 2: MiniMax lost time to local repair loops, not to broad research overhead

### What

Inside the implementation window:

- GPT-5.4 reached phase-1 code completion in **11.8 minutes**.
- MiniMax reached phase-1 code completion in **19.0 minutes**.

MiniMax also shows:

- more `bash` calls (`51` vs `35`),
- far more `edit` calls (`25` vs `6`),
- twice as many `go test` cycles (`14` vs `7`),
- and additional `go build`, `gofmt`, and `golangci-lint` cycles that GPT does not show in the same window.

The failure snippets explain the delay:

- unused import / test compile issue
- wrong-directory `go test`
- duplicate YAML `flags` schema bug
- `RootDir` / empty-root test failures
- alias query-field validation gap
- wrong-directory `git commit`
- exhaustive / ineffassign lint cleanup

### Why it matters

This is the main answer to the timing question. MiniMax did not spend the extra time on architecture discovery or docs. It spent the extra time **repairing a large package burst until all tests/lint/commit checks passed**.

### Review interpretation

The process issue is not “MiniMax doesn’t know what to build.” The process issue is:

> MiniMax compressed too much phase-1 work into one package burst, then paid back the time as local stabilization/debugging.

### Fix

For MiniMax 2.7:

1. commit each sub-slice earlier (types, parse SQL, parse alias, compiler, catalog, assets),
2. run focused `go test ./pkg/minitracecmd/...` after each slice,
3. checkpoint the phase boundary explicitly (annotation or task check) before moving on,
4. only then do the diary/docmgr bookkeeping.

### Evidence

- Timing: `scripts/12-phase1-timing.sql`
- Tool usage: `scripts/04-tool-frequency-preboundary.sql`
- Build/test churn: `scripts/06-build-cycle-preboundary.sql`
- Failure snippets: `scripts/13-bash-failures-preboundary.sql`

## Finding 3: MiniMax’s final phase-1 code is not low quality overall

### What

MiniMax’s package has several positive qualities:

- significantly larger and more exhaustive test files,
- richer commentary in some source files,
- a stronger alias parser in one important respect (`query:` is explicitly captured and rejected),
- a richer `framework-summary.sql` than GPT’s corresponding built-in command.

### Why it matters

The report should avoid the false conclusion that “MiniMax took longer, therefore the code must be worse.” That is not what the evidence says.

The more accurate assessment is:

- **process was rougher**,
- **boundary completeness was narrower**,
- **but some file-level quality aspects are genuinely stronger**.

### Review interpretation

This is why the right recommendation is process tuning and scope discipline, not a wholesale redesign.

### Fix

Preserve the good parts of MiniMax’s style:

- stronger edge-case tests,
- explicit parser guardrails,
- richer command docs/examples,

while changing the delivery path:

- smaller slices,
- earlier validation,
- explicit phase checkpoints.

### Evidence

- MiniMax test sizes and parser code:
  - `/home/manuel/workspaces/2026-04-08/sqleton-minitrace-minimax/go-minitrace/pkg/minitracecmd/parse_alias.go`
  - `/home/manuel/workspaces/2026-04-08/sqleton-minitrace-minimax/go-minitrace/pkg/minitracecmd/parse_alias_test.go`
  - `/home/manuel/workspaces/2026-04-08/sqleton-minitrace-minimax/go-minitrace/pkg/minitracecmd/parse_sql_test.go`
  - `/home/manuel/workspaces/2026-04-08/sqleton-minitrace-minimax/go-minitrace/pkg/minitracecmd/compiler_test.go`
- GPT comparison files:
  - `/tmp/gpt-phase1-2wA3Rx/pkg/minitracecmd/parse_alias.go`
  - `/tmp/gpt-phase1-2wA3Rx/pkg/minitracecmd/parse_sql_test.go`

## Finding 4: MiniMax 2.7 should use explicit phase boundaries and smaller validated slices

### What

The current comparison was greatly improved by adding explicit archive annotations for:

- `phase-1-code-complete`
- `phase-1-bookkeeping-complete`

That same idea should be used during future implementation work, not only during post-hoc review.

### Why it matters

If the model (or operator supervising it) marks boundaries as it goes, later continuation sessions and post-mortem reviews become much easier:

- no ambiguous phase cutoff,
- no accidental comparison against later work,
- faster recovery after interruptions,
- better ability to detect “stuck in repair loop” states.

### Recommended MiniMax 2.7 operating pattern

1. Start from the phase task list.
2. Implement one narrow slice.
3. Run focused validation.
4. Commit the slice.
5. Annotate the commit/tool-call as a boundary if it is a meaningful checkpoint.
6. Only then expand scope.

### Concrete example for this repo

For `GMT-002`, the ideal sub-slices are:

1. types + source detection
2. SQL parser
3. alias parser
4. compiler
5. catalog
6. embedded assets helper + smoke test

That sequence mirrors GPT’s effective process more closely than MiniMax’s one-shot phase-1 package burst.

## Overall verdict

### On the code at the end of phase 1

**MiniMax phase-1 code is acceptable and in places quite good, but it is not as complete a phase-1 endpoint as GPT’s annotated phase-1 code.**

### On why MiniMax took longer

**MiniMax took longer because it stabilized a larger burst of code through repeated local repair cycles (tests, path handling, parser edge cases, lint cleanup, and one wrong-directory mistake), whereas GPT reached phase-1 completion through smaller validated slices.**

### On what to do for MiniMax 2.7

Do **not** throw away the implementation direction. Instead:

- keep the same architecture direction,
- preserve the strong parser/test instincts,
- but force narrower slice boundaries,
- validate after each slice,
- and mark phase boundaries explicitly so the next continuation can stay phase-correct.

## Open questions

1. Should the team define GPT phase 1 more narrowly as “parser+catalog only,” or keep the current boundary that includes the embedded-assets step?
2. Should MiniMax’s missing `assets.go` / `assets_test.go` be treated as a phase-1 incompleteness bug, or as a boundary-definition mismatch?
3. Does the `compiler.go` nil-flag handling difference matter in practice, or is it only a style/API-shape difference?
4. Should the final ticket deliverable include the detached-worktree comparison paths, or should those be converted into stable excerpts/copied notes for durability?
