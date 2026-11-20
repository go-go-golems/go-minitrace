---
Title: V2 persona transcript review
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
Summary: Independent review of all 32 matrix runs, final D host replays, access observations, and separate zero-tool probes.
LastUpdated: 2026-11-17T18:39:08.489523-05:00
WhatFor: Distinguish actual API defects from persona mistakes and unsupported report claims.
WhenToUse: When interpreting the V2 persona grades and deciding follow-up scope.
---

# Task A: all eight completed runs

Reviewed only runs with final receipts containing `elapsed_seconds`, under `/private/tmp/gmt015-persona-v2-matrix/runs/matrix/A-*`. Read answers, reports, saved SQL/JS, command receipts, stdout and relevant stderr. Parsed stored output as data; did not execute historical commands, rebuild the binary, or modify the frozen bundle.

All eight runs produced nine distinct identities with **793 actual turn rows and 706 actual tool rows**. All preserve Codex session `01a1364a-f58a-7ce3-8460-a856dacc8a1a` as **7 turns, 0 tools**. Independently normalized SQL and JS output projections agree in every run and agree with `answer.json`. The 16 principal command receipts have exit code 0 and matching stdout hashes. This corroborates equality beyond the evaluator's self-reported `sql_js_equal` boolean.

The evidence below is relative to each run's `receipts/` directory. Suffix `.json` denotes a command receipt and `.stdout` the corresponding output.

| Run suffix (after A-) | SQL receipt | JS receipt | Interpretation |
| --- | --- | --- | --- |
| deepseek-4.1-flash-high-r1 | sql-nine | js-nine-b | Same ID/count output; later JS source adds framework and metric cross-check fields. |
| deepseek-4.1-flash-high-r2 | sql-counts | js-counts2 | Actual `turn_rows`/`tool_rows` agree; stored rollups are extra cross-checks. |
| deepseek-4.1-flash-low-r1 | sql-nine | js-nine | SQL envelope is complete with nine rows; JS uses `SessionToolCounts()` and bound-by-identity turn aggregation. |
| deepseek-4.1-flash-low-r2 | sql_counts | js_counts_final | Complete SQL envelope; final strict JS counts `TurnRows()` and `SessionToolCounts()` and independently compares grouped SQL. |
| glm-5.3-flash-high-r1 | sql-counts | js-counts | Actual correlated counts match on all identities. |
| glm-5.3-flash-high-r2 | sql-per-session | js-run | Count projections agree; framework field is aliased differently, so entire objects are not byte-identical. |
| glm-5.3-flash-low-r1 | sql-run | js-run2 | Correlated counts match; JS checks error/completeness. |
| glm-5.3-flash-low-r2 | sql-sessions3 | js-counts | SQL JSON and JS ASCII table agree after parsing the table; no claim of byte-identical output is warranted. |

## Product behavior and persona mistakes

No mandatory task-A product blocker was established. Both DeepSeek low runs exercise the new `SessionToolCounts()` recipe directly, including zero-tool identity. The other six JS commands issue SQL via `db.execute({sql})` or `db.query`; these confirm CLI/Goja integration but are not independent database implementations.

DeepSeek low r2 first ran strict `TurnRows()` with a 16,000-byte cell limit and received a nonzero incomplete-evidence error. Its diagnostic run recorded 793 rows, no row-overflow flag, and eight clipped content/raw_json cells (18,012–36,741 original bytes). Raising the cell limit to 200,000 while retaining strict mode made the final run succeed. This is intended strict-evidence behavior, with recoverable diagnostics, not lost turns or a count bug. Aggregating counts in SQL would avoid fetching these large cells.

Repeated initial failures are authoring/CLI discovery errors: nonexistent `turns.id` or `sessions.framework`; missing scanner metadata or SQL `short`; flags attached before a leaf command loads; unsupported `--cache-mode` on JS command leaves; positional arguments to `validate` instead of `--path`; and attempts to overwrite create-only receipt names. Reports generally preserve these failures and successful retries. They motivate discoverability improvements, but do not demonstrate incorrect query results.

GLM low r2 retains a broken optional `query-repo/counts/session-counts.sql` referring to `t.id`, and its separate tool-only SQL omits zero-tool sessions by grouping `tool_calls`. Its successful ad hoc SQL receipt and JS command both anchor on `sessions` and preserve the missing-zero case correctly. Thus the final counts have evidence, while the report's list of runnable SQL artifacts overstates the validity of that particular saved file.

GLM high r2's report says **710 tools**; its table, answer and successful outputs sum to **706**. This is a report arithmetic error, not an engine discrepancy. DeepSeek high r1 modified its JS source for the later cross-check; the final file no longer produces the exact original `js-nine-b` projection, although its final source and cross-check receipt still demonstrate the same actual counts.

## Optional history observations: keep scope bounded

Several reports interpret empty `history file-history` output as possible missing file evidence. These are not established regressions in the new query contracts. The existing command first selects coarse matches from paths, command text and arguments, applies `LIMIT`, and only then filters structural path candidates in JS (`pkg/minitracecmd/core/history/file-history.js`, SQL around lines 143–161 and filtering around 186–196). Small limits on broad fragments can therefore exhaust the candidate budget before surviving path matches, which is a plausible explanation for DeepSeek low r2's `.md`/`md` empty results versus `SKILL.md` hits. Its limits also differ (5/3 versus 10), so the reports do not isolate a substring-matching defect. Treat this as a pre-existing bounded-query/discoverability follow-up, not a demonstrated evidence-fidelity blocker.

A null turn field in the optional failure preset was reported without a source-level attribution check. No finding about conversion fidelity should be based solely on that report. The task-A evidence supports normalized identity/count correctness; it does not establish native transcript completeness, exact replay, or every optional history semantic.


# Task B: seven completed runs reviewed

Coverage at this checkpoint: all four DeepSeek runs, GLM high r1, and GLM low r1/r2. GLM high r2 lacked a completed receipt and was excluded. All seven final synthetic answers agree with observed output: UTF-8 length 4; clipped text `ab` with `complete:false`; strict error with zero rows; archive ordinals `[0,1,2]`; default ToolRows ordinals `[1,2,0]`; one duplicate and one conflicting session group. Saved JS actually calls the respective APIs rather than embedding those expected answers.

Principal evidence under each run's `receipts/`: DeepSeek high r1 `synthetic-final`; high r2 `probe-synthetic3` plus `manifest-three`; low r1 separate `utf8`, `permissive`, `strict`, `archive-order`, `default-order`, `manifest`; low r2 `fidelity2`; GLM high r1 `btask1`; low r1 `js-b2`; low r2 `synth-checks2`. Inspected synthetic receipts/output hashes agree. Synthetic and real summaries remain separate; each reviewed answer includes a real identity/tool count and token evidence from each framework.

Expected negatives are not execution failures: JS can return an error envelope as successfully observed data (strict probe wrapper exit 0), while direct strict CLI execution rejects before result emission. DeepSeek high r1 initially used its 3-byte handle for ToolRows too, clipping IDs to `a-s`, `m-t`, `z-f`; the final probe uses an unclipped handle and records complete rows with full IDs. That is a correct recovery, not an ordering defect.

Two report interpretations need correction:

- **GLM low r1's extension diagnosis is unsupported and contradicted by receipts.** Its first `js-b` invocation points `--archive-glob` to `RUN/qc/../fixtures/archive.json`, not the bundle's actual fixtures directory. It gets an empty ToolRows result. Renaming/copying to a local `.minitrace.json` happens to fix the path as well. GLM low r2 successfully loads the actual absolute bundle `fixtures/archive.json` with RuntimeArchives. Thus `.json` is not proven unsupported. The unmatched-input/empty-DB behavior may deserve a separate fail-fast review, but this run does not establish an extension bug.
- **SessionSummary is singular.** DeepSeek high r2's surprise that it returns only one row follows its explicit `LIMIT 1` implementation. Use SessionToolCounts or a sessions query for enumeration. Its custom-binding complaint was worked around, but other B runs successfully use bound sections; no general binding failure is established.

Token totals are sums of observed normalized fields, not independent provider billing totals. Reports appropriately distinguish `recorded_usage` from unknown zero-fill. The large Codex session-metrics/turn-usage discrepancy noted by DeepSeek high r1 remains an attribution limitation requiring native evidence; it is not by itself a newly demonstrated conversion bug. GLM high r1's statement that clipping implicitly exercises row overflow is too broad: cell clipping and row-limit overflow are separate checks, and its report later correctly leaves row overflow pending.

# Task C: three completed runs reviewed

Coverage at this checkpoint: DeepSeek low r1/r2 and GLM low r1. Remaining C runs lacked completed receipts at inspection and are not graded here. Read saved runnable JS and final answers; independently recomputed every reported raw-file SHA-256, byte count, physical-line count and `type == api-request` count for all nine sources in all three answers. All match, with whole-source scope. The real Claude counts are 107, 0 and 7; every Codex/Pi file has zero records of that specific type. Synthetic request counts remain separate at 2 complete / 1 incomplete.

Primary evidence: DeepSeek low r1 `receipts/scan-real9-v2` and `scan-fixture`, with `qrepo/evidence/native-scan.js`; DeepSeek low r2 `receipts/scan3` and `detail1`, with `repo/native-scan/{scan,detail}.js`; GLM low r1 `receipts/ns2`, with `cmds/native-scan.js`. GLM returns JSON text inside the formatter's `value` field rather than a direct structured row; its final answer nevertheless contains the full hashes and correct observations. Its primary receipt hash was verified.

Native API shape use is correct: callbacks inspect `record.value.type`, inspection accesses `requests[].status` and `scan`, and detailed compaction projections use `boundary.source`, `boundary.uuid`, nested summary/message coordinates and optional-neighbor guards. DeepSeek low r2's detail script specifically exercises the documented nested compaction shape. DeepSeek low r1 also records an early callback stop as `consumed_prefix`, keeping it distinct from EOF results in the mandatory answer.

The malformed synthetic line, missing-blob request and early-stop prefix are expected negative evidence, not scanner failures. The fixture's 13 physical lines versus `wc -l` reporting 12 follows its final unterminated line. CLI command nesting/flag placement caused recoverable authoring errors, with successful leaf receipts retained. No reviewed native API misuse or mandatory C product blocker was found.

These runs support byte/record fidelity and explicit replay limitations. Zero Claude-style request records in Codex/Pi must not be interpreted as no instructions; the reports acknowledge this. `exact_replay_possible:false` is supported by missing request dependencies and the documented file-local chronological index limitations; whole-source hashing alone never establishes semantic replay completeness.


## B/C completion checkpoint: all sixteen reviewed

The remaining B GLM high r2 and five remaining C runs have now completed and been reviewed, bringing coverage to **8/8 B and 8/8 C**. B high r2's `qrun1`, `qrun2`, `qrun3` receipts support clipping/strict, the two orderings, and manifest facts. Its final real-session command aggregates the recipe SQL and forwards `recipe.args()`, correctly avoiding large text cells. Earlier missing bind arguments and oversized output were generated-probe defects/expected guardrails, not failures of the final evidence contract.

All eight C answers independently match the raw files for all nine hashes, byte counts, physical-line counts, and api-request counts (72 source/run comparisons). Remaining scripts use valid ForEach/Inspect shapes. Some inspect request completeness only for the fixture; that meets the task and does not justify a stronger claim about completeness of individual real requests. C GLM low r2 constructs its own display envelope with one serialized row but `count` equal to the number of scanned files; this is a generated presentation inconsistency, not a QueryResult engine defect. Its extracted mandatory scan values are correct.

**Overwritten evidence distinction:** C GLM high r2 has seven early stdout streams whose current bytes no longer match their receipt hashes (`scan-pi-2026-09-29T1`, `scan-pi-2026-09-17T1`, `scan-claude-code-agent-ae7076`, `scan-claude-code-7dce992c-779`, `scan-pi-2026-09-15T1`, `scan-claude-code-d047a697-35c`, `scan-codex-rollout-2026`). Its report acknowledges redirecting onto captured outputs and receipt-name collisions. Those early streams are not intact evidence. All nine replacement `receipts2/scan2-*` streams match their hashes and support the final answer. No stdout hash discrepancies were found among the inspected command receipts of the other seven C runs. This is an evidence-handling mistake in the generated workflow, not a native scanner integrity failure.


# Final D review: all eight terminated runs

Final review coverage is **32/32 matrix runs (8 each A/B/C/D)**, including the unsuccessful D run. Seven D runs produced final answers and all seven original host replays passed their three selected real-source queries plus help: 28 exit-zero replay commands, expected session identities and counts, `--source`/`--archive-glob` visible, bad.js skip diagnostics retained. Replay stdout hashes match. Literal negative JS metadata defaults remain unsupported (`unary_expression` diagnostic); recognizing that limitation is the intended negative test, not a failed healthy sibling.

**D GLM high r1 remains a task failure:** exit 143 after the 480-second timeout, no `answer.json`, no `report.md`, and consequently no original host replay. Its partial checkpoint says mandatory observations were collected but final answer/report remained pending. Last activity pursued optional schema/write-rejection probes instead of finalizing. Worker `final-good-claude-code`, `final-good-codex`, `final-good-pi` receipts succeeded, but those do not turn an unfinished run into a passing final submission. This is a prioritization/completion failure; no product failure is established by the timeout.

Generated code and explanations require separate judgment from successful replay. D GLM low r1's claim that metadata markers must be at the top is contradicted by successful final commands with markers at the bottom. D DeepSeek low r1 attributes duplicate `fields` flags to section metadata plus siblings, but its original `good(fields)` parameter was not mapped by the differently named metadata binding; other D runs successfully combine sections and the bad sibling. Its broad diagnosis is unproven. Unsupported expression metadata, mismatched parameter/section bindings, wrong paths and command nesting explain multiple failed probes. Large-cell clipping is an expected completeness guardrail; raising a limit or aggregating small SQL projections can recover without dropping evidence checks.

## Additional unscored zero-tool probe

Ran the pinned binary against every saved D command using its original per-run sandbox policy and the zero-tool Codex archive `01a1364a-f58a-7ce3-8460-a856dacc8a1a`. The reproducible host script is `scripts/persona-testing-v2/zero_tool_probe.py`. It uses fixed argv, does not execute worker shell text, and stores separate immutable command receipts/stdout/stderr under bundle `control/zero-tool-probes/matrix/`. Original answers, host replay and grades were unchanged. Compact results are checked in at `sources/persona-v2-zero-tool-probes.json`.

| D run suffix | Separate zero-tool result | Attribution |
| --- | --- | --- |
| deepseek-high-r1 | Correct identity / 0 | LEFT JOIN preserves session identity. |
| deepseek-high-r2 | Correct identity / 0 | LEFT JOIN preserves session identity. |
| deepseek-low-r1 | Exit 0, empty array | Generated GROUP BY over ToolRows has no session row when tools are absent. |
| deepseek-low-r2 | Correct identity / 0 | Resolves session separately and counts bound SQL rows. |
| glm-high-r1 | Correct identity / 0 | **Unfinished generated code** only; still a timeout/no-answer mandatory failure. |
| glm-high-r2 | Correct identity / 0 | Resolves session independently of ToolRows length. |
| glm-low-r1 | Exit 1, no tool rows | Generated code treats zero tools as an error. |
| glm-low-r2 | Exit 1, expected one session, got 0 | Generated identity map is built solely from tools. |

Thus **4/7 final-answer commands** preserve zero-tool identity; one additional unfinished command also does, giving **5/8 saved commands**. Three failures are generated-command edge defects, not `SessionToolCounts()` defects: the new standard recipe is not used in these failing commands. D's mandatory selected-source checks used nonempty sessions and remain unchanged; this edge probe broadens coverage without retroactively redefining the score.

## Access audit and confidentiality limits

Inspected recorded read/bash tool arguments across all 32 runs for explicit other-run paths, evaluator/private-oracle references, host `agent-analysis` artifacts, control/grade paths and parent-relative traversal; inspected the relevant returned directory-listing output. No recorded content read of another run's answer, private oracle, host analysis summary, control grades or zero-tool results was identified. This is a trace-based observation, not an exhaustive filesystem access proof.

There **was filename exposure**: D DeepSeek low r1 ran a broad bundle `find` followed by `ls public agent-analysis private-oracle control`. The result listed host `agent-analysis/summary.json`, `sql.stdout`, archive/output names and control names including `matrix-grades.json`. Traversal into sibling run directories and private-oracle returned `Operation not permitted`. Subsequent recorded calls did not read the exposed host/control contents. C DeepSeek low r1 also ran a bundle-wide markdown filename search while excluding private-oracle. Parent-relative paths in B/D probes otherwise targeted mistaken corpus paths, not sibling answers.

Do not claim exhaustive prevention of other-answer reads or full filesystem confidentiality. The frozen policy protected enumerated paths, and host analysis/control read denial was incomplete during this cohort. Filename exposure alone does not demonstrate answer contamination. Any subsequent hardening must be described as a post-cohort change, without rewriting this evidence history.
