---
Title: Persona testing findings and improvement plan
Ticket: GMT-015
Status: active
DocType: reference
Topics: [analysis, minitrace]
Summary: Distinguishes demonstrated product defects, interface friction, agent mistakes, and evaluator defects using executed persona traces.
---

# Persona testing findings and improvement plan

The persona runs demonstrate useful analytical work with the new APIs, but they also expose gaps that a field-correctness score alone cannot capture. The strongest remaining product concern is inconsistent evidence visibility across interfaces: JS envelopes disclose loss, while permissive CLI JSON prints only the surviving rows. Separately, the published TypeScript result declaration disagrees with actual early-error values. Several expensive failures are command discovery and schema-navigation problems rather than broken native reconstruction.

This is a review of completed artifacts and selected execution traces, not an assertion that every run or every model conclusion was independently replayed. All 32 runs and the eight independent command replays are complete. `sources/persona-matrix-results.json` contains the current completed-run counts and v2 raw field grades; `sources/persona-adjudications.json` preserves scope ambiguities and evaluator corrections. `reference/07-persona-test-results.md` describes the scoring and selected B/C audits. `reference/08-persona-friction-review.md` is the companion A/D review; the private per-run `independent-replay.json` receipts are the authoritative replay evidence.

All paths below beginning with a run identifier are relative to `/private/tmp/gmt015-persona/matrix`. Raw transcripts and full private outputs stay outside the ticket. Source files under test were not repaired during evaluation.


<!-- final-matrix-metrics:start -->
## Completed experiment results

All **32 matrix runs** finished with exit code zero and produced both answer.json and report.md. There were **zero timeouts**. They emitted **1,671 tool calls** against a frozen product implementation. The separate successful GLM/low pilot is excluded from these totals, as is its earlier failed provider-registration startup.

The real corpus contains nine historical sessions: three Claude Code, three Codex and three Pi, with 793 normalized turns and 706 tool records. Recorded activity spans September 15–November 12, 2026. Inputs were selected for framework/size/shape diversity, not randomly sampled.

| Model | Requested thinking | Finished runs | Median seconds | Tool calls | Literal A/B/C checks |
|---|---|---:|---:|---:|---:|
| deepseek-4.1-flash | high | 8 | 206.6 | 502 | 140/140 |
| deepseek-4.1-flash | low | 8 | 128.6 | 353 | 140/140 |
| glm-5.3-flash | high | 8 | 264.1 | 484 | 134/140 |
| glm-5.3-flash | low | 8 | 190.2 | 332 | 136/140 |

The literal evaluator reports **550/560 A/B/C fields**. These are exact answer-contract checks, not a percentage of tasks successfully completed. Ten mismatches concern underspecified dataset/order fields: agents sometimes supplied real or combined request counts instead of fixture counts, or chronological/real-archive ordinals instead of fixture archive order. Supporting computations and separate substantive errors are adjudicated in sources/persona-adjudications.json. No model ranking is justified by this tiny, fixed-order, paired-persona study. Wall times include tool execution and overlapping Go processes; requested thinking is recorded in Pi metadata, not independently verified backend reasoning behavior.

All **eight D commands** were source-reviewed and independently replayed. All four DeepSeek-generated variants preserve all nine session identities; all four GLM-generated variants omit the zero-tool session despite keeping the correct aggregate total of 706. Every variant enforces its required source flag and allows a valid command beside the malformed signed-default sibling. An additional reviewer probe using an unmatched source exposes one GLM-generated empty-result crash; valid framework invocations still succeed. This extra edge probe is not conflated with the original requested task.

All 32 persisted Pi sessions were converted and queried through go-minitrace itself. Native, normalized and runner tool/failure accounting and the five compared usage fields match, with no reported warnings. The source/input integrity check passes all 324 hash checks and all 32 model/thinking metadata checks. This verifies unchanged known inputs and product files; it does not prove workspace-scope compliance.

That distinction matters: two agents created 12 untracked log/output files in the repository through relative shell redirects. The files were preserved under /private/tmp/gmt015-persona/scope-artifacts and removed from the product worktree by moving them, not deleting evidence. The manifest is sources/persona-scope-violations.json. The next harness should use a hash-pinned binary from each run directory and enforce a read-only repository, rather than depending on prompts alone.

The ticket preserves 127 matrix-generated source artifacts plus the pilot script, the task runner, independent graders, reviewed replay controls, corpus provenance and aggregate receipts. Raw transcripts and streamed model outputs remain private. A durable local compressed evidence archive is recorded in sources/persona-private-evidence.json under ~/.local/share/go-minitrace/experiments/GMT-015; temporary-file cleanup will not erase the only retained copy. All-run event scanning and targeted deep review are distinguished explicitly; this is not a claim that every transcript line was manually reviewed.

<!-- final-matrix-metrics:end -->
## What the executed tasks establish

The first audited B run implemented UTF-8 measurement, cell clipping, strict rejection, row overflow, parameterized SQL, recipe execution, manifest identity checks, source ordering, and token provenance in runnable JavaScript. Its output artifacts show that `abé` is four UTF-8 bytes, clipping to three bytes emits `ab` with explicit incompleteness, and strict mode returns zero partial rows. Later B runs reproduced these results. This is positive evidence for the core fidelity implementation, not proof that every surface preserves its diagnostics.

The first audited C run streamed all nine real native files with the new reader. Independent Python hashing and counting matched each file's hash, byte size, physical record count, and whole-source scope. It also emitted a consumed-prefix receipt for early stop and real diagnostics for malformed and oversized records. Synthetic request resolution produced two complete requests and one unresolved reference. Real request inspection correctly retained missing-blob uncertainty rather than fabricating instruction bodies. These checks support file-local evidence inspection; they do not establish exact causal replay or universal provider-format support.

Independent D replays show the invalid signed-default sibling no longer prevents valid commands from loading. They also expose a different problem: all four reviewed GLM-authored count queries excluded the zero-tool session. The aggregate tool total still matched 706, concealing the missing identity. This is a caller query bug with a reusable lesson for recipes and examples: validate per-session coverage, not only sums.

## Prioritized findings

| Priority | Owner / category | Demonstrated behavior | Small proposed change |
|---|---|---|---|
| P1 | API gap: CLI evidence visibility | Permissive CLI JSON drops the JS-style completeness envelope | Add an explicit envelope output option and warn/document row-only fidelity limits |
| P1 | Product: declaration contract | QueryResult declares array rows/columns, early errors serialize null | Normalize empty arrays or declare nullability consistently; add type/runtime contract test |
| P1 | Harness: answer scope | Count/order keys underspecified despite correct computations | Next cohort names fixture/real counts and archive/chronological orders separately |
| P1 | Agent query correctness | Correct total of 706 hides eight instead of nine sessions | Left-join from sessions, count nullable IDs, verify identities and zero-tool rows |
| P2 | Docs / command ergonomics | Flags repeatedly attempted on the wrong command surface | Add a short scope matrix with exact runnable examples and contextual errors |
| P2 | Docs / schema discoverability | Agents guess token_usage or positional rows, then recover through recipes | Document recipe output columns and object-row shape; expose simple CLI schema discovery |
| P2 | API diagnostics | execute(rawSQL) fails through recipe.sql contract; malformed archives misclassified | Add argument-shape hints and field-level archive decode context |
| P2 | Harness: path interpretation | v1 rejects valid corpus-relative scan paths | Fixed in evaluator v2; retain original checks and correction record |
| P3 | Native projection usability | Caller guesses compaction properties and emits nulls | Copyable projection example and generated declaration sample checked against fixture |

Priority means review/implementation order for this project, not a security severity assessment.

## Verified API behavior and the confirmed declaration bug

### Permissive CLI results lose the evidence envelope

In `B-deepseek-4.1-flash-high-r1`, the JS probe calls:

```javascript
const db = mt.db().File(input).SQLiteMemory()
  .MaxCellBytes(3).Build();
const result = db.queryResult("SELECT 'abé' AS text");
```

`out/fidelity-clip.json` retains `complete:false`, `truncated:true`, and the cell diagnostic `originalBytes:4`, `emittedBytes:2`. The equivalent permissive `query run --max-cell-bytes 3` output in `out/cli-clip.json` is only a row containing `text:"ab"`. The separate row-overflow CLI artifact similarly contains the remaining rows without `rowsTruncated`. `--strict-evidence` correctly rejects the incomplete result, so existing recoveries are strict mode, JS envelopes, or a separate CLI `--run-record` receipt containing completeness and cell diagnostics. The CLI can preserve that evidence today; the gap is its absence from permissive row-only stdout.

The gap is concrete: a consumer cannot distinguish a complete `ab` from a clipped `abé` from permissive CLI JSON alone. Preserve current row formatting for compatibility, but add an explicit result-envelope mode. Its acceptance test should invoke the actual CLI and assert both clipped and row-limited diagnostics survive JSON serialization. A documentation warning is an immediate fix, not a substitute for a structured fidelity-preserving output path.

### TypeScript promises non-null arrays that runtime errors do not supply

`pkg/minitracejs/typescript.go:34` declares `columns: string[]` and `rows: Array<Record<string, unknown>>`. Early paths in `pkg/minitracedb/query.go` return `QueryResult{Error: ...}` with nil arrays. Executed invalid SQL in B outputs confirms `rows:null`, `columns:null`, and `complete:false`; strict incomplete results instead contain `rows:[]`.

This is a real declaration/runtime mismatch. A consumer following the type and reading `result.rows.length` can fail while trying to handle a query error. Prefer one consistent result shape with empty arrays on every error; alternatively explicitly declare nullability and document guarding. Test both read-only validation failure and missing-table failure against the emitted JS result and generated declarations. Do not treat a caller's missing null check as the whole cause while the public type promises otherwise.

## Friction that should improve without widening the API unnecessarily

Repeated A attempts use `convert claude-code` or `convert pi` with `--run-record` and receive `unknown flag`; the Codex converter does support that flag, so this is a subcommand-specific distinction, not a blanket conversion limitation; repeated query-command attempts place `--cache-mode` or `--max-rows` on a JS command surface that does not accept them. The A/D reviewer reproduced these flag-scope distinctions. Query-run flags, runtime archive flags, builder policy, and conversion receipt behavior need a compact command matrix. Error hints should point to the correct surface rather than requiring the caller to infer it from generic help.

For example, explain that runtime archives select inputs for a JS command, while `.SQLiteMemory().MaxRows(...)` controls that command's database handle. Do not introduce aliases for every flag immediately: first make the existing separation legible and provide one working invocation per surface.

Several B agents guess a physical `token_usage` table. Their attempts fail and they recover by using `mt.query().TokenUsageRows()` or inspecting its SQL. Others index `db.query()` rows as arrays and produce null-valued projections before switching to column-name properties. The maintainable fix is to document row objects explicitly and generate or test recipe column references. Include the important semantic distinction: tool estimates live in a numeric token-total field but carry `estimated_tokens` as the unit; a column-name guess must not replace checking origin and unit.

The A/D reviewer also reproduced `execute(rawSQL)` failing with a recipe-shape TypeError. The method intentionally takes a recipe or `{sql,args}` object; the API need not silently accept every input. A targeted error such as “execute expects a recipe or {sql,args}; use queryResult(sql, ...args) for a SQL string” would shorten recovery while preserving the contract.

A malformed pilot normalized archive with `metrics:[]` was reported as unsuitable native JSON/automatic-conversion-disabled input. The caller first tried a filename change, which did not address the malformed field. Distinguish decoding errors in a recognized normalized archive from unknown native formats and include the field/type mismatch. This finding concerns the synthetic pilot setup and diagnostics, not real corpus corruption.

## Caller mistakes that must not become product bug claims

The reviewed D GLM variants lose zero-tool sessions through different implementations: a left join followed by a non-null tool-ID filter, an inner join, or enumeration from `Object.keys(recipeCounts)`. Each yields eight sessions with the unchanged total of 706. The fix is an explicit sessions-driven result:

```sql
SELECT s.session_id, COUNT(tc.tool_call_id) AS tool_count
FROM sessions AS s
LEFT JOIN tool_calls AS tc ON tc.session_id = s.session_id
GROUP BY s.session_id;
```

A meaningful acceptance check compares the full identity set and each count, including zero. Do not change the API merely because a caller selected a tool-only universe.

C GLM/high/r1 reads guessed compaction fields such as `c.boundary.logicalParentUuid`; the available value is `c.logicalParentUuid`. Its wrapper also discards metadata, summaries, and boundary coordinates through wrong property names. The saved nulls therefore are projection errors, not absent native evidence. A later projection should use `c.metadata`, `c.summaries`, `c.beforeMessage`, and `c.boundary.source`; a tested example can prevent this without inventing another abstraction.

B GLM/high/r1 explains UTF-8 length correctly in its computed answer but incorrectly says JavaScript `.length` for `abé` is four. It is three UTF-16 code units. It also describes empty queryOne results as indistinguishable from broken queries after demonstrating that broken queries throw. Review the prose separately from the computed values.

Do not accept hypotheses as diagnoses. The A/D reviewer disproved the claim that `RuntimeArchives()` works inline but fails when called from a helper; a controlled example supported both. The actual failure involved invoking a command group instead of its nested leaf. Likewise B GLM/high/r2's report alleges silent repository disappearance after a multi-verb edit but explicitly says it never isolated the construct. That is a reproduction candidate, not a confirmed recurrence of the old catalog-blast defect. The successful invalid-sibling replays establish a narrower fact and do not prove every scanner error path is good.

## Harness findings and fair interpretation

The frozen B prompt requests archive and chronological order but labels the answer simply `source_ordinals`. B GLM/high/r1 computes both correctly and chooses chronological order for the answer. Preserve its raw mismatch and identify answer-scope ambiguity rather than claiming misunderstanding. B GLM/high/r2 instead explicitly reports the contiguous ordinals of a real Pi archive (0 through344); review that as dataset-scope selection, not automatically as an ordinal calculation bug.

The C prompt requests both synthetic and real inspection but explicitly scopes only the hash to the fixture. C GLM/high/r1 and low/r1 provide real counts at top level while computing the correct fixture 2/1 in their outputs. Preserve raw contract scores and this ambiguity together. High/r1 also has a separate genuine reporting error: it gives real incomplete total106 even though its per-file breakdown is106+1=107. Low/r1 reports the correct real 7/107.

Evaluator v1 additionally misresolved valid `framework/basename` paths against its own cwd. Evaluator v2 corrects only that interpretation, accepts exact unique corpus-relative or absolute paths, and continues rejecting missing identities, duplicate canonical paths, and ambiguous basenames. The original 3/42 C GLM/low score and 42 checks remain in adjudications; corrected score is40/42, with count-scope mismatches retained. This was an evaluator defect, not model failure.

For a subsequent cohort, use an explicit answer schema with `synthetic_request_counts`, `real_request_counts`, `archive_source_ordinals`, and `chronological_source_ordinals`, and state the path forms accepted. Ask agents to save intermediate answers early, then expand evidence coverage. Keep artifact completion, independently checked fields, reviewer replay checks, and reported model usage as separate measurements. None is a substitute for the others.

## Small implementation sequence

1. **Contract consistency:** resolve QueryResult nullability and add CLI envelope output with focused runtime/CLI tests. Document compatibility behavior.
2. **Discovery and diagnostic quality:** add the command-scope matrix, recipe column references, object-row example, tested compaction projection, and contextual errors for execute input shape and malformed normalized archives.
3. **Evaluation revision:** name answer scopes, preserve zero-row session examples, retain intermediate artifacts, and carry independent review annotations next to raw scores. Run a fresh cohort rather than editing the current prompt mid-experiment.

Native exact replay, cross-file inheritance recovery, and direct script-runner ergonomics can remain separate proposals. The current evidence supports improving the existing boundaries first. No conclusion here requires accepting model self-evaluation as proof or presenting all API behavior as correct.

## Breadth of transcript review and its limits

The broad review uses `scripts/persona-testing/audit_traces.py` to inspect every emitted tool-call event available at the snapshot, including argument text, explicit tool-error events, attempted API features, and candidate scope violations. The B/C review snapshot covers **14 finished B/C runs**, not every line of all 32 runs manually. That was an intermediate snapshot. The final all-run event audit covers all 32 finished runs; the snapshot count is preserved here only to describe the bounded deep-review pass. The private snapshot is `/private/tmp/gmt015-persona/bc-review-trace-audit.json`; the final complete-matrix audit is /private/tmp/gmt015-persona/trace-audit.json.

Those 14 completed traces contain nine events explicitly marked as tool errors. They include run-local cleanup permission errors, missing guessed fixture paths, an edit whose old text did not match, and JSON-reading failures after prior commands produced non-JSON or empty output. These are not nine product bugs. Conversely, zero flagged tool errors does not establish zero command failures: shell pipelines or deliberate error-capture probes can return transport success while the contained program prints an error.

A second bounded pass therefore searched text returned by tool events for diagnostic signatures. It found repeated `unknown flag: --output`, incompatible cache-directory/memory-mode arguments, guessed `token_usage` tables, nonexistent `agent_framework` columns in a projection, missing recipe SQL arguments, disallowed `sqlite_master`/`pragma_table_info` introspection, and wrong command nesting. These are diagnostic occurrences, not deduplicated incident counts: the same failure may be printed again in a report or observed deliberately. In particular strict-incomplete errors and invalid-table tests are requested demonstrations, so their presence is positive coverage rather than a failed task.

Manual deeper review concentrates on the first DeepSeek/high B/C runs, GLM/high and low scope mismatches, and the later GLM/high repetition. It reads actual authored functions and saved result structures alongside execution events, rather than trusting debrief assertions. For B GLM/high/r2, the345-element ordinal answer is corroborated by the Pi tool-fidelity artifact and explicitly scoped to that real archive in its prose. For C GLM/high/r2, the final real 7/107 counts coexist with correct fixture 2/1 outputs. Both raw oracle mismatches are preserved with structured scope annotations. The latter run eventually saves raw compaction diagnostics that recover coordinates discarded by its initial guessed projection; a review should recognize that recovery instead of treating its first null output as the final result.

Wall time also limits interpretation. B GLM/high/r2 finished in472.546 seconds and C GLM/high/r2 in450.091 seconds against a480-second allowance; neither is a timeout. They did substantially more discovery and repair work than their final answer fields reveal. Do not infer that higher requested thinking caused the additional work, or rank models from these few heterogeneous runs. All 32 final receipts record zero timeouts. Keep that denominator separate from field correctness and preserve this rule for future cohorts.

## Concrete acceptance checklist for follow-up changes

A focused product patch should be reviewable with small fixtures and direct assertions:

- Run the CLI with `SELECT 'abé' AS text` at three bytes. Envelope mode must retain `complete:false`, emitted text `ab`, and original/emitted byte counts4/2. Existing row mode remains compatible and documented.
- Query an invalid table and reject a non-SELECT statement through both JS result surfaces. Declarations and runtime must agree about rows/columns on both paths; a type-checked caller should not require an undocumented null guard.
- Build a three-session database containing one zero-tool session. The published per-session count example must return all three IDs, including a zero count, and match independent source arrays.
- Test the documented command nesting and flag locations by executing copied examples. Verify normal help shows the required flag and runtime archive selectors, and invalid argument hints identify the correct surface.
- Test a normalized archive with an intentionally wrong field type. The error must mention that field/type and distinguish malformed normalized input from unsupported native format.
- Execute the compaction projection example against the tiny fixture. Assert boundary source, logical parent, preserved metadata, summary list, preceding message, and following assistant, rather than merely checking the object exists.

These checks should accompany the respective focused patches. They do not require a broad interpreter rewrite, provider replay engine, or another expensive model matrix before each small change. A later matrix can measure whether the improved diagnostics reduce repair steps, while keeping correctness evaluation independent of the agents' own confidence or final verdicts.

## Prompt and harness changes for the next cohort

Use a small explicit JSON schema whose names identify the dataset and order: synthetic.request_counts, real.request_counts, archive_source_ordinals and chronological_source_ordinals. State whether real totals include the fixture, name the accepted path base, and require an identity list alongside aggregates. These changes address demonstrated ambiguity; they must be evaluated in a new cohort, not retroactively inserted into this one's prompts.

Separate mandatory acceptance checks from optional exploration, and require intermediate artifacts before long debriefs. Two high-thinking runs finished at 450 and 473 seconds under the 480-second limit; a successful final answer does not mean the workflow is efficient. Record shell exit status directly instead of masking errors through echo/cat pipelines. Prebuild the tested binary once, record its hash and revision, and keep Go caches fixed. This also avoids the observed fresh-HOME dependency-cache side effect.

Enforce filesystem permissions for the repository and evaluator, not merely a prose prohibition. Preserve attempted violations and classify them independently from product correctness. A future documentation experiment can compare help-only discovery, a short task-oriented quickstart, and the full reference using the same inputs and explicit answer schema. That would measure which documentation helps, rather than asking models to retrospectively guess what influenced them.
