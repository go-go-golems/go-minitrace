---
Title: 'Session Analysis: minimax vs gpt-5.4'
Ticket: MINIMAX-VS-GPT-COMPARE
Status: active
Topics:
    - code-review
    - analysis
    - minimax
    - gpt
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/analysis/archive/active/2026-04/2d525241-fe32-417b-8576-b29ce3b3e47c.minitrace.json
      Note: MiniMax converted session with synced phase-boundary annotations
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/analysis/archive/active/2026-04/7f61f412-40f0-417f-ab85-4dffdb9927e5.minitrace.json
      Note: GPT converted session with synced phase-boundary annotations
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/scripts/03-phase1-boundary-events.sql
      Note: Detects candidate phase-1 completion events from transcript tool calls
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/scripts/03-phase1-boundary-events.sql:Finds candidate phase-1 completion events from tool calls
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/scripts/10-annotation-boundaries.sql
      Note: Queries synced phase-boundary annotations from the archive
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/scripts/10-annotation-boundaries.sql:Queries synced phase-boundary annotations from the archive
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/scripts/12-phase1-timing.sql
      Note: Measures implementation-window timing
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/scripts/12-phase1-timing.sql:Measures implementation-window timing
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/scripts/14-compare-phase1-tree.sh
      Note: Compares the GPT phase-1 tree against the MiniMax phase-1 tree
    - Path: ttmp/2026/04/08/MINIMAX-VS-GPT-COMPARE--compare-minimax-vs-gpt-5-4-implementation-approaches-sqleton-minitrace/scripts/14-compare-phase1-tree.sh:Compares the GPT phase-1 code tree against the MiniMax phase-1 tree
ExternalSources: []
Summary: Minimax vs GPT-5.4 comparison restricted to the end-of-phase-1 code state and the implementation window leading to it, using go-minitrace archives plus synced boundary annotations.
LastUpdated: 2026-04-09T01:05:00Z
WhatFor: Explain the state of the code at the end of phase 1 and why MiniMax took longer to reach it.
WhenToUse: Read this before the findings doc if you want the raw evidence and the phase-boundary method.
---


# Session Analysis: MiniMax vs GPT-5.4 at the End of Phase 1

## Executive summary

This analysis compares **MiniMax-M2.7** and **GPT-5.4** only up to the **phase-1 code-complete boundary** for `GMT-002` (`MinitraceCommand` / sqleton-style query loading). The comparison is grounded in a shared ticket-local minitrace archive, with synced tool-call annotations marking the exact phase-1 code-complete and bookkeeping-complete events for both sessions.

The current evidence supports three conclusions:

1. **GPT-5.4 reached phase-1 code completion faster**: about **11.8 minutes** from the implementation prompt, versus **19.0 minutes** for MiniMax.
2. **MiniMax lost time mostly to local repair loops**, not to broad exploration or documentation overhead. Its implementation window shows substantially more `edit`, `go test`, `gofmt`, `go build`, and `golangci-lint` churn focused on a small set of package files and tests.
3. **MiniMax’s phase-1 code is not low-quality in the sloppy sense**, but its resulting package is **narrower and less complete** than GPT’s phase-1 endpoint. MiniMax compensates with heavier tests/comments in several files, but it is missing some package-completeness pieces that GPT already had by its annotated phase-1 boundary.

## Scope and method

### What is in scope

- `GMT-002` implementation work only
- comparison at the **end of phase 1**
- the implementation path from the prompt:
  - `Add detailed tasks to the ticket, then work on them one by one, committing as you go. Keep a frequent diary.`
- code quality of the resulting phase-1 package
- process analysis explaining why MiniMax took longer

### What is explicitly out of scope

- GPT work after phase-1 code completion
- GPT render / CLI integration (`afeb0a4`, `b218017`) as code-review targets
- later MiniMax doc commits except as bookkeeping evidence

### Boundary definition

I marked the phase boundaries directly in the archive with synced tool-call annotations:

| Run | Phase-1 code complete | Phase-1 bookkeeping complete |
|---|---|---|
| GPT-5.4 | tool call targeting commit `7cc5370` (`Add embedded MinitraceCommand assets`) | later docmgr checkpoint after task 18 |
| MiniMax | tool call targeting commit `5bf8958` (`feat(minitracecmd): Phase 1 — parser, compiler, catalog, tests, and core commands`) | later docmgr / upload bookkeeping commit |

The implementation window is measured from the user turn containing `Add detailed tasks to the ticket...` to the `phase-1-code-complete` annotation target.

## Source sessions and archive

### Input sessions

| Run | Session ID | Source JSONL |
|---|---|---|
| MiniMax | `2d525241-fe32-417b-8576-b29ce3b3e47c` | `~/.pi/agent/sessions/--home-manuel-workspaces-2026-04-08-sqleton-minitrace-minimax--/2026-04-09T00-23-06-562Z_2d525241-fe32-417b-8576-b29ce3b3e47c.jsonl` |
| GPT-5.4 | `7f61f412-40f0-417f-ab85-4dffdb9927e5` | `~/.pi/agent/sessions/--home-manuel-workspaces-2026-04-08-sqleton-minitrace--/2026-04-09T00-13-39-925Z_7f61f412-40f0-417f-ab85-4dffdb9927e5.jsonl` |

### Archive facts

Both sessions were converted into:

- `analysis/archive/active/2026-04/2d525241-fe32-417b-8576-b29ce3b3e47c.minitrace.json`
- `analysis/archive/active/2026-04/7f61f412-40f0-417f-ab85-4dffdb9927e5.minitrace.json`

Top-level session counts from conversion:

| Run | Model | Turns | Tool calls |
|---|---|---:|---:|
| MiniMax | `MiniMax-M2.7` | 124 | 131 |
| GPT-5.4 | `gpt-5.4` | 192 | 269 |

These are full-session numbers. The more important process comparison below uses the narrower implementation window.

## Implementation-window timing

Measured via `scripts/12-phase1-timing.sql`.

| Run | Implementation start | Code complete | Bookkeeping complete | Minutes to code complete | Minutes to bookkeeping complete |
|---|---|---|---|---:|---:|
| GPT-5.4 | `2026-04-09T00:23:59.920Z` | `2026-04-09T00:35:47.880Z` | `2026-04-09T00:36:00.745Z` | **11.8** | **12.0** |
| MiniMax | `2026-04-09T00:23:23.428Z` | `2026-04-09T00:42:23.905Z` | `2026-04-09T00:45:33.488Z` | **19.0** | **22.2** |

### Interpretation

The raw result is clear: **MiniMax took about 7.2 minutes longer to reach phase-1 code completion**. The rest of this document explains where that time went.

## Tool usage inside the implementation window

Measured via `scripts/04-tool-frequency-preboundary.sql`.

| Run | bash | read | write | edit |
|---|---:|---:|---:|---:|
| GPT-5.4 | 35 | 26 | 18 | 6 |
| MiniMax | 51 | 21 | 16 | 25 |

### Interpretation

The high-level signature differs sharply:

- **GPT-5.4**: more balanced between `bash`, `read`, and `write`, with relatively few `edit` calls.
- **MiniMax**: markedly more `edit`-heavy and still more `bash`-heavy, which is the classic sign of local repair loops inside a small cluster of files.

This already points away from “MiniMax took longer because it explored more” and toward “MiniMax took longer because it had to repair the package more.”

## File-touch concentration inside the implementation window

Measured via `scripts/05-file-touch-preboundary.sql` and `scripts/07-rewrite-preboundary.sql`.

### GPT-5.4’s main touch pattern

Inside the implementation window, GPT still edited the diary a fair amount, but its code file touches are relatively shallow and wide:

- `reference/01-investigation-diary.md` — 5 edits
- then mostly one-touch writes/reads across the new package files
- one edit to `pkg/minitracecmd/catalog.go`
- one write each for the new package files

### MiniMax’s main touch pattern

MiniMax’s touches are much more concentrated in a small number of package files and tests:

- `pkg/minitracecmd/parse_sql_test.go` — **6 edits**
- `pkg/minitracecmd/catalog.go` — **4 edits**
- `pkg/minitracecmd/compiler_test.go` — **3 edits**
- `pkg/minitracecmd/parse_sql.go` — **3 edits**
- `pkg/minitracecmd/parse_alias.go` — 2 edits
- `pkg/minitracecmd/parse_alias_test.go` — 2 edits
- repeated writes to `catalog.go` and `parse_sql.go`

### Interpretation

GPT’s phase-1 implementation looks like a sequence of smaller planned slices. MiniMax’s phase-1 implementation looks like a single larger package burst that had to be debugged in place.

That difference matters because the timing question is not just “who used more tools?” It is “who had to revisit the same files repeatedly before the code boundary passed?” On that measure, MiniMax clearly had more concentrated churn.

## Build / test / formatting churn inside the implementation window

Measured via `scripts/06-build-cycle-preboundary.sql`.

| Run | go-test | go-build | golangci-lint | gofmt | git-commit |
|---|---:|---:|---:|---:|---:|
| GPT-5.4 | 7 | 0 | 0 | 0 | 11 |
| MiniMax | 14 | 2 | 2 | 2 | 3 |

### Interpretation

This is one of the strongest explanations for the elapsed-time difference.

- **GPT-5.4** spent the implementation window doing smaller commit-sized increments with fewer local repair cycles.
- **MiniMax** spent the implementation window repeatedly cycling through tests/lint/format/build before its final successful squash commit.

The extra `go test`, `go build`, `gofmt`, and `golangci-lint` traffic strongly suggests that MiniMax’s extra time was spent on package stabilization rather than on higher-level research or design.

## Concrete MiniMax failure evidence

Measured via `scripts/13-bash-failures-preboundary.sql` and direct inspection of `bash` tool-call output.

The MiniMax implementation window contains several concrete failure loops that GPT does not show in the same way inside its implementation window:

1. **Initial test compile failure** in `parse_sql_test.go`
   - unused import / broken test file state
2. **Wrong-directory test invocation**
   - `pattern ./pkg/minitracecmd/...: directory prefix pkg/minitracecmd does not contain modules listed in go.work or their selected dependencies`
3. **Alias YAML schema bug**
   - panic: `duplicated key 'flags' in struct minitracecmd.MinitraceCommandSpec`
4. **RootDir / path-handling test failures**
   - `EmptyRootDir` / `file does not exist`
5. **Alias query-field validation gap**
   - `expected error, got nil` for `TestParseAliasSpec_CannotSetQuery`
6. **Wrong-directory commit attempt**
   - `fatal: not a git repository (or any of the parent directories): .git`
7. **Lint loop**
   - `missing cases in switch of type minitracecmd.SourceKind: minitracecmd.SourceUnknown (exhaustive)`
   - `ineffectual assignment to err (ineffassign)`

### Interpretation

This is the best current answer to the user’s timing question:

> **MiniMax took longer because it had to debug and repair its package locally before the final successful phase-1 commit passed.**

The extra time was not primarily spent on diary writing or architecture exploration. It was spent on:

- broken tests,
- YAML/schema edge cases,
- path/root normalization,
- lint cleanup,
- and one wrong-directory git mistake.

## Phase-1 code state at the boundary

Measured via `scripts/14-compare-phase1-tree.sh` and direct file reads.

### Tree size and inventory

| Run | Files in `pkg/minitracecmd` phase-1 tree | Total lines |
|---|---:|---:|
| GPT-5.4 phase-1 (`7cc5370`) | 17 | 1051 |
| MiniMax phase-1 (`5bf8958`) | 14 | 1798 |

### Files GPT has at its annotated phase-1 boundary that MiniMax does not

- `pkg/minitracecmd/assets.go`
- `pkg/minitracecmd/assets_test.go`
- `pkg/minitracecmd/types_test.go`
- `pkg/minitracecmd/core/timing-analysis.sql`

### Important reading of that difference

This does **not** mean GPT’s code is automatically “better” in every file. It means GPT’s phase-1 package is **more complete at the package boundary**:

- it already exposes the embedded command catalog via `EmbeddedSourceRoot()` / `LoadEmbeddedCatalog()`
- it already has a smoke test for that embedded path
- it already includes a small types/source-kind test file
- it already includes a third built-in command (`timing-analysis.sql`)

MiniMax’s phase-1 package, by contrast, stops at the parser/compiler/catalog/core-fixture layer.

## Code-quality comparison of the phase-1 package

### Where MiniMax looks good

1. **Heavier tests and more explanatory comments**
   - MiniMax’s package is substantially longer overall (1798 lines vs 1051), largely because its tests are much more extensive.
   - `catalog_test.go`, `compiler_test.go`, `parse_sql_test.go`, and `parse_alias_test.go` are all much larger in MiniMax.
2. **Some docs/examples are richer**
   - Example: MiniMax’s `core/framework-summary.sql` is richer and more descriptive than GPT’s corresponding file.
3. **The final package still passes `go test ./pkg/minitracecmd/...`**
   - so the end state is not broken or half-checked-in.

### Where GPT looks better at the package boundary

1. **Better package completeness**
   - GPT includes `assets.go` + `assets_test.go`, which means phase-1 ends with a loadable embedded catalog path instead of only raw `core/` fixtures.
2. **Better lightweight coverage of the foundational types layer**
   - GPT includes `types_test.go`; MiniMax does not.
3. **More error context in some places**
   - GPT’s phase-1 `catalog.go` and `parse_alias.go` wrap some errors with `github.com/pkg/errors`, preserving path/context detail that MiniMax’s current code sometimes drops.

### Specific code-shape differences worth noting

#### `catalog.go`

- GPT phase-1:
  - keeps `RootDir == ""` normalization inside the loader
  - preserves `Readonly: root.Readonly`
  - wraps root-loading and alias errors with extra context
- MiniMax phase-1:
  - has more explanatory comments
  - simplifies some control flow for the linters
  - hardcodes `Readonly: true` during compile in this path
  - returns `ErrAliasTargetNotFound` without the extra alias/target context GPT preserves

#### `compiler.go`

- GPT phase-1:
  - preserves `nil` entries when normalizing optional bool flags
- MiniMax phase-1:
  - drops `nil` flags in the normalization loop
  - adds a dedicated `normalizeFolder()` helper, which is a good response to its test failures

#### `parse_alias.go`

- GPT phase-1:
  - simpler parser
  - no explicit `query:` key capture/rejection at parse time
- MiniMax phase-1:
  - explicitly adds `Query string \`yaml:"query"\`` to the intermediate YAML struct and rejects it
  - this is a **real improvement** in defensive parsing behavior

### Overall code-quality verdict at the end of phase 1

The fairest current verdict is:

- **MiniMax phase-1 code is credible and reasonably high-quality**, especially in test thoroughness and some defensive parsing details.
- **But MiniMax phase-1 is narrower and less complete as a package deliverable** than GPT’s annotated phase-1 endpoint.
- So the result is **not “bad code,” but “a slower path to a slightly less complete phase-1 boundary.”**

## Why MiniMax took longer

Based on the evidence above, the most defensible explanation is:

1. **MiniMax attempted a larger one-shot package implementation** and then stabilized it.
2. That larger burst produced **several local breakages**:
   - test compile failures
   - YAML schema issue
   - root/path handling failures
   - alias query validation gap
   - lint issues
   - wrong-directory command mistake
3. Those breakages caused **high local edit churn** in a few test/code files.
4. GPT-5.4, by contrast, moved through **smaller commit-sized slices**, reaching a phase-1-complete package boundary without the same density of visible repair loops.

The process difference is therefore best described as:

- **GPT-5.4**: more linear / staged
- **MiniMax**: more bursty / repair-heavy

## Caveats

1. The GPT implementation-window session is part of a longer earlier research/design session, so the final report should avoid comparing full-session totals directly.
2. The implementation-window start for MiniMax is taken from a split-turn continuation and is slightly less clean than GPT’s single user prompt.
3. The overlap query is only a helper. The authoritative code-state comparison comes from the phase-1 tree diff, not from normalized transcript file paths alone.
4. The failure queries are best used qualitatively; raw failure-class counts are noisier than the timing/churn/tree evidence.

## Scripts used so far

- `scripts/01-convert-sessions.sh`
- `scripts/02-session-list.sql`
- `scripts/03-phase1-boundary-events.sql`
- `scripts/04-tool-frequency-preboundary.sql`
- `scripts/05-file-touch-preboundary.sql`
- `scripts/06-build-cycle-preboundary.sql`
- `scripts/07-rewrite-preboundary.sql`
- `scripts/08-cross-session-file-overlap.sql`
- `scripts/09-docmgr-events.sql`
- `scripts/10-annotation-boundaries.sql`
- `scripts/11-phase1-start-turns.sql`
- `scripts/12-phase1-timing.sql`
- `scripts/13-bash-failures-preboundary.sql`
- `scripts/14-compare-phase1-tree.sh`
- `scripts/15-failure-counts-preboundary.sql`

## Bottom line

If the question is:

> **What is the state of affairs at the end of phase 1, and why did MiniMax take longer?**

then the current answer is:

- GPT-5.4 reached a **more complete phase-1 package boundary** (embedded loader/test + extra built-in command) in **11.8 minutes**.
- MiniMax reached a **credible but narrower phase-1 package** in **19.0 minutes**.
- MiniMax’s extra time is best explained by **implementation-local debugging and stabilization loops**, not by excessive exploration or documentation overhead.
- MiniMax’s resulting code is **not low quality overall**; in several files it is actually more heavily tested or more defensive. The issue is **speed and package completeness at the chosen phase-1 boundary**, not a total lack of engineering quality.
