---
Title: Diary
Ticket: ADD-HISTORY-QUERY-COMMANDS-2026-07-20
Status: active
Topics:
    - query-commands
    - js
    - embedded-catalog
    - skills
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Implementation diary for moving the file-history/ticket-timeline/context-window JS query verbs from the go-minitrace-transcript-analysis skill's external query-repository into go-minitrace's embedded command catalog."
LastUpdated: 2026-07-20T14:03:50.943885466-04:00
WhatFor: "Contemporaneous record of embedding the three verbs, testing the rebuilt binary, and updating the skill to stop distributing them separately."
WhenToUse: "Read before adding further embedded query commands, or before touching pkg/minitracecmd/core/history/."
---

# Diary

## Step 1: Investigate the embedded catalog mechanism

### Prompt Context

**User prompt (verbatim):** "Work in /home/manuel/workspaces/2026-07-20/add-skill-commands which contains go-minitrace checkout and actually move th skill verbs into the go-minitrace binary so that they don't have to be distributed separately. Create a new docmgr in there (which will put it into go-minitrace/ttmp and then get to work, keep a diary and commit at appropriate interval wile you work. Then update the skill"

This follows directly from ticket `GOGO-MINITRACE-HISTORY-VERBS-2026-07-20` (claw-stuff repo), which designed, implemented, and validated three go-minitrace JS query verbs (`file-history`, `ticket-timeline`, `context-window`) and shipped them in the `go-minitrace-transcript-analysis` skill's `query-commands/history/` directory, loaded via `--query-repository`. This ticket's job: stop distributing them as an external repository and make them part of the `go-minitrace` binary itself.

### What I did

Explored `/home/manuel/workspaces/2026-07-20/add-skill-commands/go-minitrace` (a go.work multi-module workspace alongside `glazed` and `go-go-goja` checkouts; git remotes `origin` = `go-go-golems/go-minitrace`, `wesen` = personal fork; clean working tree at `b813c91`).

Found the embedded-catalog mechanism:

- `pkg/minitracecmd/assets.go` — `//go:embed core` embeds the entire `pkg/minitracecmd/core/` directory tree as an `embed.FS`; `LoadEmbeddedCatalog()` builds the catalog from just that one `SourceRoot` (`Readonly: true`).
- `pkg/minitracecmd/core/{overview,files,nightly,timing,tools}/` — today's embedded commands, all `.sql` (plus one `.alias.yaml`). No JS examples exist in `core` yet, but nothing in the loader is SQL-only.
- `pkg/minitracecmd/catalog.go`'s `LoadCatalog` walks every `SourceRoot` generically via `fs.WalkDir` + `DetectSourceKind` (`.sql` → `SourceSQLCommand`, `.js`/`.cjs` → `SourceJSCommand`, `.alias.yaml` → `SourceYAMLAlias`) and dispatches to `ParseSQLCommandSpec` / `ParseJSCommandSpecs` / `ParseAliasSpec` identically regardless of whether the `fs.FS` is `embed.FS` or an on-disk directory. **JS commands are fully supported in the embedded catalog already** — nothing in the compiler/scanner is external-repository-specific. This makes the port a straightforward file copy, not a new code path.
- `pkg/minitracecmd/assets_test.go` — `TestLoadEmbeddedCatalog` asserts `len(Commands) >= 9` and spot-checks specific `ByName`/`ByPath` entries for existing commands. No fixed/exact count assertion to break; will add matching assertions for the three new commands.

Also found: `~/go/bin/go-minitrace` (the binary on `PATH`, `go-minitrace version dev`) — `make install` runs `go generate ./... && go install $(CMD_DIR)`, so rebuilding via that target updates the live CLI in place, no separate deploy step needed.

Also found: this repo already bundles its own copy of the skill at `skills/go-minitrace-transcript-analysis/` — but it is a **stale, diverged baseline** (missing `references/attribution.md`, `references/js-query-authoring.md`, and everything from this campaign's work), not kept in sync with the live skill at `~/.claude/skills/go-minitrace-transcript-analysis` (confirmed by diff). No install/sync mechanism in the Makefile connects them. Decision: focus "update the skill" on the live skill (the one this whole campaign has built and the one actually loaded by Claude/Codex/Pi via the hardlinked mirrors); make one small non-scope-creeping fix to the repo-bundled copy too, since leaving it telling users to pass `--query-repository` for verbs that are now built in would be a real, easily-avoided inaccuracy.

### Why

Confirming the embedded catalog treats JS identically to SQL *before* writing anything avoids discovering a JS-specific limitation mid-implementation — cheap to check, expensive to find out the hard way after the diary already claims "just copy the files."

### What worked

Reading `catalog.go` end-to-end before touching anything paid off: the port really is "copy 3 files + add test assertions," not a new integration.

### What warrants a second pair of eyes

The decision to fix only the one stale instruction in the repo-bundled skill copy rather than fully re-syncing it with the live skill — a full re-sync is out of scope for this ticket (it's a go-minitrace binary change, not a skill-content overhaul) but leaves the repo-bundled copy still missing `attribution.md`/`js-query-authoring.md`/the interpretation lessons. Flagged, not fixed, here.

### What should be done in the future

Consider a Makefile target (`make sync-skill` or similar) that copies the canonical skill source into this repo's `skills/` directory, so the two never diverge silently again.

## Step 2: Implement, test, install, verify, commit

### Prompt Context

Continuation of Step 1 — executing the plan against real code, no new user prompt.

### What I did

1. Copied the three JS files verbatim from `~/.claude/skills/go-minitrace-transcript-analysis/query-commands/history/` into `pkg/minitracecmd/core/history/` — no changes needed to the files themselves.
2. `go build ./...` and `go test ./pkg/minitracecmd/...` passed unmodified — confirming Step 1's read of `catalog.go` was correct.
3. Wrote a throwaway test (`zz_dump_test.go`, deleted after use) to print the actual compiled `Path`/`Name` for the three new commands before writing real assertions, rather than guessing the JS single-verb-collapse path format: `history/file-history`, `history/ticket-timeline`, `history/context-window` (no `.js` suffix — the compiler collapses a single-verb file whose verb name matches the file stem).
4. Added six assertions to `assets_test.go` (`ByName` + `ByPath` for all three) matching the existing style used for the `nightly/*` commands.
5. `go test ./...` (full repo) — all green.
6. `make install` — ran the Dagger web build then `go install ./cmd/go-minitrace`, updating `~/go/bin/go-minitrace` (the binary already on `PATH`) in place.
7. Verified live: `go-minitrace query commands history --help` lists all three with **no `--query-repository` needed** (its default is now empty — previously this flag was required to point at the skill's directory). Ran all three verbs against a real converted archive (`gogowm-analysis`, from an earlier campaign) with zero extra flags — `file-history` reconstructed a real timeline, `context-window` returned a real 49-tool-call window, `ticket-timeline` correctly returned zero events for a ticket ID that ticket wasn't actually created in that session (true negative, not an error).
8. Committed (`311102e`, branch `task/add-skill-commands`): `pre-commit` hooks ran the full test suite + `golangci-lint` (0 issues) automatically via lefthook — both passed on the first attempt.

### Why

Verifying against a real archive (not just `--help`) before committing matters — the embedded-catalog *loading* mechanism working doesn't by itself prove the JS *runtime* (goja sandbox, `require("minitrace")`, `mt.sql.*` helpers) behaves identically when the source bytes come from an `embed.FS` walk instead of an on-disk `--query-repository` directory. It does — no visible difference in output between the two loading paths.

### What worked

The whole port was friction-free: no code changes to the verbs, no loader changes, tests and lint passed on the first try. This is what "the embedded catalog treats JS and SQL identically" (Step 1's finding) predicted.

### What didn't work

Nothing failed. This step is a clean confirmation of Step 1's investigation, not a debugging log — recorded plainly rather than manufacturing an artificial "what didn't work" entry.

### What I learned

`make install` triggers a real Dagger container build for the web frontend (`go generate ./...` → `cmd/go-minitrace/cmds/serve/frontend`) before `go install` — slower than a bare `go build`, but it's the correct target since it's what actually updates the live `~/go/bin/go-minitrace` binary matching how this project is normally built.

### What warrants a second pair of eyes

None beyond Step 1's open item (repo-bundled skill copy staleness).

### What should be done in the future

Same note as Step 1 about a skill-sync Makefile target.

### Code review instructions

`git show 311102e` — four files: three new `.js` under `pkg/minitracecmd/core/history/`, one test-assertion diff. Re-run `go-minitrace query commands history file-history --help` and confirm no `--query-repository` flag is required to see it.

### Technical details

Commit: `311102e` on branch `task/add-skill-commands`. Binary verified: `~/go/bin/go-minitrace` (`go-minitrace version dev`).

## Step 3: Update the skill(s)

### Prompt Context

Continuation — user's final instruction was "Then update the skill."

### What I did

1. Discovered that the "live" skill (`~/.claude/skills/go-minitrace-transcript-analysis`, hardlinked to `~/.codex/skills` and `~/.pi/agent/skills`) has its `query-commands/history/*.js` files ALSO hardlinked identically across all three mirrors (same inode `66639881` for `file-history.js` confirmed via `stat -c '%i %n'`) — the whole directory tree is kept in sync this way, not just `SKILL.md`. Removing a hardlinked file only drops that one directory entry, not the underlying inode's other links, so I removed the three `.js` files from all three mirror paths explicitly (`rm` under `~/.claude`, `~/.codex`, `~/.pi/agent`), then `rmdir`'d the now-empty `history/` subdirectory in each.
2. Edited `~/.claude/skills/go-minitrace-transcript-analysis/SKILL.md`'s "Ready-made query commands" section (renamed to "Built-in query commands") to drop the `--query-repository` flag from the example invocation and state the binary/commit dependency plainly, with a fallback note for anyone on an older `go-minitrace` build. This edit propagated to the other two mirrors automatically (same hardlink behavior verified earlier this session).
3. Added a short, accurate addition (not a rewrite) to the go-minitrace repo's own bundled `skills/go-minitrace-transcript-analysis/SKILL.md` — its "embedded catalog currently includes examples such as" list now includes the three `history` verbs with a pointer to this ticket. Correction to Step 1's assumption: that file never actually claimed `--query-repository` was required for these verbs (it predates them entirely, having no `history` mention at all) — there was nothing inaccurate to fix, only something missing to add.

### Why

Point 1 matters because a naive "edit SKILL.md and done" would have left the now-dead `query-commands/history/*.js` files sitting in all three live skill mirrors, silently drifting from the embedded-in-binary source of truth the moment anyone touched one copy — exactly the failure mode this whole ticket exists to eliminate.

### What worked

The hardlink-mirror pattern (established and documented last campaign) made both the doc edit (one edit, three mirrors updated) and the cleanup (three explicit removals, one per mirror, since removal doesn't follow the same one-edit-many-effects behavior as in-place editing) behave exactly as expected — no surprises.

### What warrants a second pair of eyes

Whether removing the skill-local copies is the right call for portability: anyone whose `go-minitrace` binary was installed before this session's `make install` (i.e., not rebuilt from commit `311102e`+) loses these three verbs entirely until they rebuild, with only a doc pointer (not a working fallback) telling them why. Accepted as the correct trade-off given the user's explicit goal ("so that they don't have to be distributed separately"), but noting it since it is a real, if narrow, regression window for anyone else on this machine's skill mirrors with a stale binary.

### Technical details

Removed: `{~/.claude,~/.codex,~/.pi/agent}/skills/go-minitrace-transcript-analysis/query-commands/history/{file-history,ticket-timeline,context-window}.js` (9 files, 3 mirrors × 3 verbs, all same 3 inodes). Edited: `~/.claude/skills/go-minitrace-transcript-analysis/SKILL.md` (propagates to codex/pi), `go-minitrace` repo's `skills/go-minitrace-transcript-analysis/SKILL.md` (repo-local, committed alongside the code change).

## Step 4: Replace the repo-bundled skill wholesale, prepare for PR

### Prompt Context

**User prompt (verbatim):** "copy the skill into go-minitrace as well under skills/ , then open a PR"

Supersedes Step 3's "small addition, not a full rewrite" decision — the user wants the full current skill in the repo, not a patched-up old baseline.

### What I did

`rm -rf skills/go-minitrace-transcript-analysis` then `cp -r` the entire live skill tree (`~/.claude/skills/go-minitrace-transcript-analysis/`) in. Net diff: `SKILL.md` and `references/queries.md` modified, five new files added (`references/attribution.md`, `references/js-query-authoring.md`, `query-commands/overview/session-activity.js`, four `scripts/*.sh`), `scripts/query_minitrace.sh` unchanged (byte-identical in both copies).

### Why

The old repo-bundled copy was a stale baseline predating this entire campaign (confirmed by diff in Step 1); patching one line was already a compromise made under an assumption of narrower scope. The user's explicit instruction removes that ambiguity — ship the current, maintained skill.

### What warrants a second pair of eyes

This is a full replace, not a merge — nothing in the old repo copy was checked for content not present in the live skill before deletion (the diff in Step 1 suggested the old copy was a strict subset, but that comparison was against an earlier state of the live skill, not re-verified line-by-line here). The PR diff itself is the review surface for this.

## Step 5: Push and open the PR

### Prompt Context

**User prompt (verbatim):** "copy the skill into go-minitrace as well under skills/ , then open a PR" (continued from "hold on what where the choices? i misclicked" / "push with --no-verify" after a hook blocker).

### What I did

`git push wesen task/add-skill-commands` hit lefthook's pre-push hook, which failed on `lint`: `cmd/go-minitrace/cmds/serve/embed.go:8:12: pattern all:frontend: no matching files found`. Reproduced twice. Diagnosed by running the exact same `go vet ./cmd/... ./pkg/...` (with and without `GOWORK=off`) standalone — both passed cleanly. Cross-referencing the hook's own log: `release`'s before-hook runs `go generate ./...` (a ~50s Dagger rebuild of `cmd/go-minitrace/cmds/serve/frontend`) in parallel with `lint`'s `go vet` — a race on the same generated directory, not a real problem with this diff.

Per AGENT.md ("don't try to fix errors yourself more than twice in a row, then stop") and the standing rule against bypassing hooks without explicit authorization, stopped and asked the user how to proceed (four options: `--no-verify` this once / keep retrying / fix lefthook's parallelism / user pushes themselves). User chose `--no-verify`.

Pushed to `wesen/go-minitrace` (personal fork), opened PR #30 against `go-go-golems/go-minitrace:main` via `gh pr create`, describing the change, the verification already done, and disclosing the `--no-verify` push and why (transparency for reviewers, since a skipped hook on a merge-target PR is something a reviewer should know about even though it was independently verified safe).

`gh pr create` also flagged "1 uncommitted change": `cmd/go-minitrace/cmds/serve/frontend/.gitkeep` had been deleted by the Dagger build's directory overwrite during `make install`/`go generate` runs this session — a build-artifact side effect, not an intended change. Restored with `git checkout --`.

### Why

Disclosing the `--no-verify` push in the PR body itself (not just this diary) matters: the repo owner reviewing the PR on GitHub won't see this diary unless they go looking, but they will see the PR description.

### Technical details

PR: https://github.com/go-go-golems/go-minitrace/pull/30. Branch: `task/add-skill-commands`, pushed to `wesen/go-minitrace`. Commits: `311102e`, `4e9dbe8`, `a4d99fc`.

## Step 6: Address the PR #30 code review comments

### Prompt Context

**User prompt (verbatim):** "https://github.com/go-go-golems/go-minitrace/pull/30 <- then address the code review comments in here." — followed by a handoff mid-task: "Continue, we are the big brother taking over as the work so far has been done by our little brother. Address the code review issue, assess their work and what can be salvaged, what needs to be improved, etc..."

Three automated review comments: one P1 on `file-history.js` (multi-file Codex patches lose every file after the first), two P2s (`session-activity.js` last-activity computation, and SIGPIPE exit 141 in the two `discover_*_by_cwd.sh` scripts).

### Assessment of the in-progress P1 work inherited from the earlier session

The core approach was right and was kept: replace the "only consult `arguments_json` when `file_path` is empty" gate with `extractCandidatePaths(row)`, which harvests **structurally** file-path-shaped candidates — the `file_path` column, every `*** Update/Add/Delete File:` patch header, every JSON `file_path`/`path` key, and shell redirect targets — then keeps only the candidates that actually match the search fragment. This fixes the reviewer's bug without regressing to a blanket substring search over the payload, which is what the original design existed to avoid (a `Write` call whose free-text `content` argument merely *mentions* a path is not a touch of that path). `canonicalizePath` and the per-row dedup were also sound.

Four defects were found in the handed-over state and fixed:

1. **`effectiveCommands(row)` was reading a column that was no longer selected.** An earlier edit dropped `tc.command` from the SQL SELECT after mistakenly judging it unused, so `row.command` was `undefined` on every row and the claude-code/pi branch of `effectiveCommands` was dead code. Restored `substr(COALESCE(tc.command,''),1,2000) AS command`.
2. **The WHERE clause never matched on `tc.command`.** Candidates could be extracted from `command`, but rows were only fetched when `file_path` or `arguments_json` matched — so a Bash heredoc whose target path appears only in `command` was never retrieved in the first place. Added `OR COALESCE(tc.command,'') LIKE ...`.
3. **The payload window was smaller than the review comment's own failure mode.** `arguments_json` was truncated to 8000 chars by `substr`, and the runtime `CellChars(4000)` limit cut it to 4000 before the JS ever saw it. A multi-file patch longer than that loses its later files to truncation — the exact bug being fixed, reintroduced by a different mechanism. Raised to `substr(...,1,64000)` with `CellChars(100000)` so the cell limit can never be the binding constraint.
4. **Summary grouping still split one file into several groups.** Dedup by canonical path was applied per row, but the group key was the *raw* path, so `~/x` recorded by one tool call and `/home/manuel/x` recorded by another produced two summary rows for one file. Group key is now `canonicalizePath(t.file_path)`.

`extractRedirectTargets` was also tightened to skip `&`-prefixed (`2>&1`, `>&2`) and `(`-prefixed (process substitution) targets, which name file descriptors and subprocesses rather than files.

### Verification

`file-history --path "01-diary.md"` against the claw-stuff self-referential archive: 22 timeline rows / 4 summary groups, **0 duplicate `(session_id, turn_index)` pairs**, `match_source` split 15 `file_path` / 7 `arguments_json-extracted-path`. The 7 extracted paths are the Bash heredoc diary appends recovered by redirect-target extraction — the capability that had been silently lost. The summary group count dropping from the pre-fix 10 to 4 is the canonical-grouping fix working: 4 distinct real files, previously split across 10 path spellings.

### P2: `session-activity.js` last-activity

`MAX(COALESCE(t.timestamp, tc.timestamp))` over `sessions LEFT JOIN turns LEFT JOIN tool_calls` is wrong twice. `COALESCE` is per-row and returns its first non-null argument, so once a session has any turns at all, every joined row carries a turn timestamp and tool-call timestamps are never considered — a session whose last tool call ran after its last turn reports a stale time. Joining both tables also builds a `turns x tool_calls` cross product per session.

Replaced with a `UNION ALL` of the two timestamp streams, aggregated once per session, then `LEFT JOIN`ed to `sessions`. This is also portable: it avoids scalar two-argument `MAX(a,b)`, which SQLite has but DuckDB spells `greatest`.

Honest note on verification: **the wrong-answer symptom does not reproduce on the tinyidp corpus.** Checked every session with a direct correlated-subquery comparison of `MAX(turns.timestamp)` against `MAX(tool_calls.timestamp)` — the turn timestamp is later in all 8 sessions, which is the expected shape when a session ends with an assistant reply. The fix is correct by construction rather than confirmed against a failing case. The performance half *is* measurable: 1.76s (old) vs 0.35s (new) on these 8 sessions, ~5x, driven by session `019f47f7` alone at 1639 turns x 6144 tool calls ≈ 10M cross-product rows to compute one maximum.

### P2: SIGPIPE in the discover scripts

`... | while read; do ...; done | head -n "${LIMIT}"` under `set -o pipefail`: `head` exits once it has LIMIT lines, the still-writing loop takes SIGPIPE, and the pipeline reports 141 — a failure status for a normal truncation.

Reproduced the failure directly (`seq 1 10000 | while read ...; done | head -n 3` → status 141) and confirmed the replacement returns 0. The limit is now a counter with `break` inside the loop, and input arrives by process substitution (`done < <(find ... | sort -r)`) rather than a pipe, so `pipefail` never inspects `find`/`sort`'s status when the loop stops reading early; it also keeps `count` in the current shell instead of a subshell. Applied to both `discover_codex_by_cwd.sh` and `discover_pi_by_cwd.sh`. Ran the real codex script end to end against `~/.codex/sessions` with `limit=2` on a cwd that genuinely has matches — exit 0, correct truncation.

Audited the other four scripts in `scripts/` for the same pattern; only these two had a `done | head`.

### Skill mirror sync

`session-activity.js` and both scripts also live in the three hardlinked live-skill mirrors (`~/.claude`, `~/.codex`, `~/.pi/agent`), which share one inode per file but are *separate* inodes from the repo copy. Synced with `cat repo-file > ~/.claude/.../file` — truncate-in-place preserves the inode, so one write propagates to all three. Verified afterwards that the inodes are still shared and that all three mirrors are byte-identical to the repo copy.

`file-history.js` needs no mirror sync: it now ships inside the binary and was removed from the live skill in Step 3.

### Verification of the whole tree

`go build ./...`, `go test ./...` (all pass), `golangci-lint run ./...` (0 issues), `make install` and re-ran the verbs against real archives.

## Step 7: Embed the remaining skill verbs

### Prompt Context

**User prompt (verbatim):** "which verbs are we adding, are we adding all the scripts / verbs from the skill into the binary, btw?" then "embed all the verbs from the skill."

### What the audit found

Steps 1-6 embedded exactly three verbs (`history/file-history`, `history/ticket-timeline`, `history/context-window`), joining 15 pre-existing embedded commands. One query-command file was still external: `skills/.../query-commands/overview/session-activity.js`, carrying **two** verbs (`session-activity`, `file-activity`). There was no principled reason for it to stay outside while the history verbs moved in — it is the same kind of artifact and needed the same `--query-repository` flag to run.

### What I did

Split the one two-verb file into two one-verb files rather than copying it across as-is. Every other command in `pkg/minitracecmd/core/` declares a single verb per file, which lets the catalog collapse the path (`overview/session-list`, not `overview/session-list/session-list`). Copying the combined file verbatim would have registered the awkward `overview/session-activity/session-activity` and `overview/session-activity/file-activity`.

- `pkg/minitracecmd/core/overview/session-activity.js` — `session-activity`, with the `path_contains` and `write_only` fields dropped from its section since it never used them.
- `pkg/minitracecmd/core/files/file-activity.js` — `file-activity`, filed under `files/` alongside `file-operations.sql` and `file-timeline.sql` rather than under `overview/`. A file-oriented verb sitting in `overview` was the anomaly; the CLI path was changing regardless of which group it landed in, so this was the moment to fix it. Flagged to the user in case they want it back under `overview`.

Removed `skills/.../query-commands/` entirely — the skill now ships no query commands, only prose and shell scripts.

Added four `assets_test.go` assertions (ByName and ByPath for both verbs), with a comment recording *why* the flat paths are the expected ones, so a future edit that recombines the files fails the test with an explanation rather than a bare nil check.

### A stale claim in SKILL.md, caught while editing

SKILL.md line 40 still described `file-history` as: "`arguments_json` fallback only fires when `file_path` is empty (avoids matching prose that merely mentions the path)." That is exactly the behavior the Step 6 P1 fix **removed** — the gate was the bug. Rewrote the entry to describe the structural-candidate extraction and both of its consequences (multi-file patches attribute correctly; prose still is not counted). Documentation describing a fixed bug as if it were current design is worse than no documentation, since it would send a future reader looking for a gate that no longer exists.

### Correction: the skill mirrors are symlinks, not hardlinks

Earlier steps in this ticket recorded the three live skill mirrors as hardlinked files sharing inodes, with the note that `rm` would have to be repeated per mirror. That is wrong. `~/.claude/skills` and `~/.pi/agent/skills` are **symlinks to `~/.codex/skills`** — there is one real directory and two pointers to it. The identical inodes observed earlier are explained by this just as well as by hardlinks.

The practical difference matters for deletion: `rm -rf ~/.claude/skills/.../query-commands` removed it from all three views at once, because there is only one directory. The earlier "repeat the rm per mirror" guidance would have been harmless but was based on a wrong model. Verified with `ls -ld` on all three paths.

### Verification

`go test ./...` passes (including the four new catalog assertions), `golangci-lint run ./...` reports 0 issues. `make install`'d and ran both verbs against the tinyidp archives **with no `--query-repository` flag** — `overview session-activity` returns sessions ordered by last activity, `files file-activity` returns per-(session, file) rows with operation counts. SKILL.md synced to the live skill.
