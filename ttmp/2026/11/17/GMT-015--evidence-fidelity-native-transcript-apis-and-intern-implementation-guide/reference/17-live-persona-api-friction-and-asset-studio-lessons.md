---
Title: Live persona API friction and Asset Studio lessons
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
LastUpdated: 2026-11-18T17:33:20.253191-05:00
WhatFor: ""
WhenToUse: ""
---

# Live persona API friction and Asset Studio lessons

## What the pilot established

Two analysts each investigated the same nine normalized archives from Claude Code, Codex and Pi. Each had three turns: the opening file-target ledger, a stakeholder follow-up, and a request to rerun the same command on Codex-only archives. Each ultimately produced exact ledgers with 34, 34 and one target. GLM's first host replay was blocked by a duplicate scratch command. The saved ledger passed separately, and the original failure remains part of the record.

The observed friction concentrated in the route from a JS file to a runnable command. The evidence does not establish a need to reshape the fluent query builder. Our earlier narrow matrix found specific composition problems, with a useful example and targeted diagnostic already confirmed by a GLM retest. [Fluent API matrix](15-local-js-api-acceptance-and-persona-replay-results.md) · [Historical run receipts](../sources/live-persona-pilot.json)

| Signal from the corrected pilot | DeepSeek | GLM | Interpretation |
| --- | ---: | ---: | --- |
| Unresolved command surfaced as `--archive-glob` | 4 | 15 | CLI diagnostic defect compounded by the historical wrapper's root-only invocation. The CLI and wrapper have both changed since the run. |
| Catalog skipped invalid JS declarations | 2 | 6 | Static metadata misuse. Both encountered the scanner contract while trying to discover or register commands. |
| `sqlite_master` rejected | 1 | 1 | Intended read policy. The discovery path to schema help and JS `db.schema()` did not come to hand first. |
| Model-facing tool preview shortened | 13 | 12 | Harness delivery cap. These counts do not measure SQLite truncation or incomplete queries. One DeepSeek subprocess also hit its output cap. |
| Raw source open denied | 1 | 0 | Historical pilot scope. It supplied normalized archives, so a follow-up requiring native source could not be corroborated there. |

Counts are repeated tool results within two runs and may overlap. A scanner warning and the archive flag error can occur in the same result. A valid exit and a complete result are separate facts. The 34 targets are normalized file-target records. Their evidence statuses do not all establish file effects.

## What I would fix or verify first

1. **Put a runnable command example at the discovery point.** Show a tiny root verb with matching filename and CLI name, then a grouped example where the first `__verb__` argument binds a top-level function and the metadata `name` sets the CLI leaf. Put the JS and exact invocation together. Make scanner errors name the file, the rejected registration and the next help topic. The corrected CLI already reports available children and Pi can now list the artifact repository and invoke a complete path.
2. **Make schema discovery the SQL-first path.** The schema help is supported and both agents reached for SQLite introspection first. Put the schema command beside the first SQL example and return a pointer to it in the SQLite error that SQL-only users actually see. Use `db.schema()` and `db.tables()` beside the first JS example. Keep the allowlist.
3. **Carry query evidence through the tool boundary.** Use small projections for first queries and label three distinct states: an incomplete query envelope, a shortened delivery preview, and a subprocess output failure. Preserve full local replay artifacts and include only selected cited records in the judge input where disclosure is appropriate. Choose a separate scenario with approved native artifacts before testing how well models corroborate native file effects.

For the next run, keep the same corpus, binary, scenario and thinking settings as far as practical. Measure calls before the first valid JS script, repeats of the same command-resolution error, and exact host replay after the scope change. Repeat a hard case under the same configuration before drawing a conclusion about the change. Keep API success separate from analyst narrative and model-judge preference.

## What transfers from Asset Studio

**Fix the strongest available layer.** The P3D playbook traces an awkward step to a specific cause, then puts a constraint or diagnostic where the error occurs. Its design gives `cyl()` positional-argument guidance at the call, instead of after the entire source fails compilation. The corresponding minitrace fix is the local missing-command diagnostic and a concrete discovery command, followed by better docs for scanner metadata. This favors improving the existing grammar and feedback before adding another fluent layer. [Agent prompt playbook](</Users/manuel.odendahl/code/tulip/playback/.worktrees/feat-asset-studio/docs/operations/agent-prompt-playbook.md:285>)

**Verify the evidence the judge actually sees.** Asset Studio initially judged claims about earlier views from the final viewport. Feeding it the captures associated with those claims changed what the grounding score measured. The equivalent here is passing exact replay values and, when judging narrative, the cited records. Our API-focused Codex judge received categorized signals and bounded host receipts. It can assess routing and documentation friction. It cannot independently verify narrative claims about files without the claim and its source evidence. [Asset Studio testing process](</Users/manuel.odendahl/code/tulip/playback-vault/Projects/2026/11/17/PROJECT REPORT - Asset Studio - Persona Runs and the Issues That Keep Coming Back.md:1131>)

**Replay one failing step before repeating the whole scenario.** The P3D team tried several explanations for a bad reply. Moving the state block into a system message did not help. Quoting the outstanding user request in the tool result fixed four of four replay attempts. Our equivalent is to target one specific exploratory JS call, change only the diagnostic or its help pointer, and check whether the next attempted command resolves. Then run the full stakeholder turn to test the composed workflow. [Asset Studio replay diary](</Users/manuel.odendahl/code/tulip/playback/.worktrees/feat-asset-studio/ttmp/2026/11/12/PB-ASSET-STUDIO-001--asset-studio-an-agent-driven-p3d-asset-creator-grounded-in-cameras-and-reference-images-with-a-fluent-js-api/reference/01-diary.md:2102>)

**Audit the tester and pin the round.** Asset Studio found tester errors in every reviewed round and recommended replaying the same saved transcripts across judges and repeating harder scenarios. Minitrace already exposed an oracle with the wrong operation taxonomy and a host replay blocked by a scratch-file collision. Freeze the corrected oracle and judge profile, retain original outcomes and recovered checks separately, and report per-case repeat ranges. [Asset Studio tester defects and repeats](</Users/manuel.odendahl/code/tulip/playback-vault/Projects/2026/11/17/PROJECT REPORT - Asset Studio - Persona Runs and the Issues That Keep Coming Back.md:1133>)

## Codex API review

I rejudged both saved runs with `openai-codex/gpt-6.2-sol` at medium effort using the authenticated Codex CLI. Both returned a completed turn with no tool actions recorded. Both identified command discoverability and scanner guidance as priorities. Both distinguished normalized schema policy from incomplete delivery. Their inputs are bounded counts and host checks, so their agreement corroborates the classification of friction. It does not add new measurements.

[Judge implementation](../scripts/persona-live/judge.py) · [Saved-batch rejudge](../scripts/persona-live/rejudge.py) · [Local review viewer](http://127.0.0.1:8797/)

[Codex judge receipts and verdicts](../sources/live-persona-codex-api-review.json)
