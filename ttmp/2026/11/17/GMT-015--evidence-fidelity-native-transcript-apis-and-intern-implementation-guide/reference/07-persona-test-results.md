---
Title: Persona experiment independent result grading
Ticket: GMT-015
Status: active
Topics: [analysis, minitrace]
DocType: reference
Summary: Completed experiment status, independent field grading, and limits of model-authored answers.
---

# Persona experiment independent result grading

## Current status

The final independent snapshot contains **32 planned, 32 started and 32 finished runs**, with 32 answer/report pairs and zero timeouts. Literal A/B/C field checks are 550/560; the ten scope/order mismatches are annotated separately from substantive analytical errors. All eight D commands have reviewed replay receipts. The final machine-readable result is sources/persona-matrix-results.json (evaluator v2).

The matrix crosses four task personas, two Lunaroute models, requested low/high thinking, and two repetitions. Requested thinking is a client setting, not independently verified backend reasoning behavior. Nine selected historical archives span Claude Code, Codex, and Pi. Their purposeful size/framework selection does not support population-wide model or product rankings.

## Independent oracle and scoring

`scripts/persona-testing/fixtures.py` creates synthetic normalized archives and a native JSONL file without calling go-minitrace. The private oracle is not an input for tested agents. The evaluator checks fixture hashes on every synthetic grading run. The corpus manifest is also withheld from tested agents. For actual row counts the evaluator verifies each archive's SHA256, parses the frozen archive independently, and counts its `turns` and `tool_calls` arrays; aggregate metadata is not accepted as the count oracle.

`scripts/persona-testing/summarize_results.py` reads only final receipts: `elapsed_seconds` must be present. The runner initially writes receipts before model execution, so existence alone would incorrectly grade active work. A temporarily truncated receipt is skipped and reconsidered on the next invocation.

- **A / intern:** session identity set, exact session-row cardinality, and per-session turn/tool counts are checked. Claimed SQL/JS equality, framework discovery, and conversion validation remain unverified claims until execution traces are reviewed.
- **B / analyst:** UTF-8 byte length, permissive clipped text and completeness, strict empty rows and error presence, source ordinal ordering, and duplicate/conflict counts are checked against synthetic facts. These eight fields do not establish that every requested exploratory operation was performed.
- **C / forensic reviewer:** complete/incomplete synthetic request counts and whole-file hash are checked. Each real native scan must identify the exact input path and match independently computed SHA256, bytes, physical line count, and `whole_source` scope. Duplicate/missing scan rows fail cardinality and identity checks. The answer must not claim exact causal replay is possible.
- **D / reusable-command author:** boolean fields are schema-checked and retained as provisional claims. They contribute **zero independently correct fields** until safe command replay and trace inspection establish their truth. A separate `independent-replay.json` can be retained alongside the run; the summarizer includes it without silently promoting arbitrary receipt content to a score.

The shared grader requires exact JSON types: `true` cannot pass as the integer `1`, and missing/null keys cannot pass a false boolean. Extra answer fields are ignored. A correct answer is evidence of field correctness only; it is not proof of API use, task completion, or a safe reasoning process. Runnable JS/SQL presence and report/answer presence are recorded separately.

## Reproduction

From the worktree root:

```sh
TICKET=ttmp/2026/11/17/GMT-015--evidence-fidelity-native-transcript-apis-and-intern-implementation-guide
python3 "$TICKET/scripts/persona-testing/summarize_results.py" \
  --output "$TICKET/sources/persona-matrix-results.json"
```

The default matrix, corpus manifest, and private oracle reside under `/private/tmp/gmt015-persona`. Raw model events, copied transcripts, and full answers remain there; the ticket stores compact grading receipts. The summarizer does not invoke models or execute model-authored code. Running it repeatedly is safe while the matrix progresses.

Each model/thinking/task aggregate includes started/finished runs, answer/report presence, timeout count, independently correct/graded fields, tool calls, tool error events, wall time, and reported usage. Usage fields come from runner receipts and preserve their original names; they are not independent measurements of backend computation. No numerical conclusion should collapse different task denominators into an apparent universal accuracy score.

## Validation and diary checkpoint

The synthetic grader passed an exact-answer positive check across 21 optional/base fields and rejected wrong-type, missing-key, and non-object answers. The first validation command incorrectly asserted that the schema contained 23 fields; correcting that evaluator bookkeeping assertion to 21 passed. This was a caller test error, not a go-minitrace failure.

The initial summarizer run completed successfully and reported `partial`, 32 planned, four started, zero finished. It verified the frozen normalized archive hashes while deriving row counts. Further API-friction conclusions require completed traces and direct reproduction; none are claimed from this initial snapshot.

## Completed-run checkpoint: first DeepSeek/high repetition

The next snapshot contains nine started and five finished runs out of 32 planned. All four DeepSeek/high/r1 tasks finished; GLM/high/r1 task A also finished. Independently checked answer fields were A DeepSeek **20/20**, A GLM **20/20**, B DeepSeek **8/8**, and C DeepSeek **42/42**. These are different rubrics with different denominators, not comparable task-accuracy percentages. D contributes no answer-field score. Its separate reviewer replay passed six bounded checks, including matching actual tool counts across all nine archives (706 total). The aggregate now records `reviewed_checks` and `reviewed_checks_passed` separately from answer correctness.

### B: substantial execution beyond the eight answer fields

Inspection of `B-deepseek-4.1-flash-high-r1/events.jsonl`, authored JS, and saved outputs corroborates API execution rather than an answer copied from expectations. The run made 50 tool calls. Its `qrepo/probe/fidelity.js` implements and runs UTF-8 length, permissive clipping, strict clipping rejection, separate row overflow, builder recipes, bound SQL recipes, exception/envelope contrasts, source ordering, manifest identity, and token provenance. Three `out/corpus-*.json` outputs show summaries for all three frameworks. Python calls project already-produced results or edit the run's own JS; the inspected analytical functions execute minitrace SQL and builders.

Concrete output evidence extends beyond the answer schema:

- `out/fidelity-overflow.json` distinguishes row truncation from cell clipping and records strict empty rows.
- `out/fidelity-envelopes.json` contains actual invalid-query exceptions and error envelopes, alongside builder and bound-parameter results.
- `out/fidelity-tokens.json` separates `unknown` from `recorded_usage` for absent versus explicit-zero usage, and retains estimated-tool units separately.
- `out/fidelity-toolrows.json` contrasts `[0,1,2]` archive ordinals with `[1,2,0]` chronological ordinals.
- `out/cli-clip.json` and `out/cli-overflow.json` contain only formatted rows; unlike JS envelopes, these saved CLI outputs do not disclose incompleteness. This is a concrete usability concern demonstrated by output artifacts, not merely a model opinion.

Limits and caller behavior remain relevant. The byte-clipping probe selects a SQL literal `abé` rather than reading the fixture's tool-result column; this exercises clipping correctly but does not independently prove that every archive field has identical handling. The report describes status distinctions but its fixture/real examples happen to have matching original and derived status; that distinction is not empirically demonstrated by this dataset. A failed intermediate query guessed a `token_usage` table before consulting the recipe/schema; this is caller discovery friction, not a confirmed missing SQL table defect. The cache experiment overrides `HOME`/`XDG_CACHE_HOME`; its artifacts can support the narrow cache observation, but this is not a recommended reproduction pattern. A correct final answer is therefore not blanket approval of every method or narrative claim.

### C: forensic coverage and disciplined uncertainty

`C-deepseek-4.1-flash-high-r1/repos/native-scan/native-scan.js` uses `mt.native().File(...).ForEach(...)` to stream all nine frozen real files. The independent grader checks every whole-file hash, byte count, physical line count, and scope against the actual frozen bytes. The saved early-stop result is **five lines, 1,630 bytes, `consumed_prefix`, stop reason `callback`**; it is explicitly distinguished from the whole-file hash. Synthetic malformed output identifies line 12 at byte offset 919. A separate 32-byte record-limit probe produces `record_too_large` diagnostics while retaining the whole-source scan receipt. These are real output artifacts, not inferred from the final answer alone.

The synthetic inspection demonstrates complete and incomplete request references plus compaction coordinates. Real Claude inspections retain missing-blob diagnostics; the reviewer does not reinterpret missing references as known instruction bodies. The authored JS scans other provider markers separately, while the report correctly distinguishes a provider-specific compaction marker from an exact replay boundary.

The SQL instruction-read investigation includes `out/target-read-row.json`, which preserves read arguments, a real `CLAUDE.md` text prefix, `truncated:true`, and `full_bytes:31975`, with 10,150 stored characters. `out/target-search-row.json` shows why `operation_type=READ` alone is insufficient: a shell `rg` search also receives that classification. The report distinguishes search hits, bounded shell excerpts, and actual read-shaped output. A built-in `history file-history` command was executed and its output saved. This supports the requested evidence distinction; it does not certify the exact assembled model prompt.

Some report language still needs qualification. Calling a mutable native builder “singleton-like” overstates the observed behavior: retaining configuration on one builder does not establish global singleton state. The statement that no filesystem module is exposed is not independently established by these probes and should not become a product capability claim. Likewise the flattened request `refs` list resolves message references; not repeating the base request ID there is not itself evidence that reconstruction omitted the prefix. These are reviewer corrections to model interpretation, not reasons to discard valid scan results.

### Evidence preservation and scope

Each of these B/C run directories now contains `independent-trace-audit.json`, hashing the inspected JS, final answer, report, event stream, and selected output artifacts (19 B artifacts and 14 C artifacts). This freezes the reviewed evidence without copying private transcript content into the ticket. The audits inspect saved execution events and outputs; they do not replay authored scripts. D's other reviewer separately reran safe reviewed commands and retained `independent-replay.json` with command receipts and source hashes.

The completed-run summarizer was rerun after adding separate D review aggregates. No product source was changed during this review. The experiment remains partial, and the next generated JSON snapshot may contain more completed runs than this prose checkpoint.

## Adjudication: GLM/high/r1 scope mismatches are not equivalent to analytical failures

The fixed prompt and raw deterministic scores are retained unchanged. The B run scored **7/8** because its top-level `source_ordinals` is `[1,2,0]`. However, `out/tool-order.json` contains both correct arrays: `archiveOrder` has ordinals `[0,1,2]`, and `chronologicalOrder` has `[1,2,0]`. The authored JS runs explicit SQL orderings for both. Its report states that it deliberately placed the chronological sequence in the answer, while documenting archive order separately. The task requested both orderings but described the answer field only as “source_ordinals (ordered integers)”; it did not specify which ordering the evaluator expected. **Classification: underspecified answer mapping / harness ambiguity, with correct demonstrated ordering analysis.** Do not report this as a failure to understand `source_ordinal` or an API defect.

The C run scored **40/42** because the top-level counts are 7 complete and 106 incomplete, versus the fixture oracle's 2 and 1. Its report explicitly raises uncertainty over whether those fields refer to the fixture, one real source, or all real sources. The prompt requests fixture and real inspections and explicitly qualifies only the hash as synthetic. `out/inspect-synth.json` nevertheless contains `completeCount:2` and one incomplete request; the final `answer.json.notes` also correctly records fixture counts 2/1. **Classification: underspecified count scope / harness ambiguity, with correct demonstrated fixture reconstruction.** This mismatch alone is not evidence that `Inspect()` was used incorrectly.

There is also a separate, genuine reporting error in C: its intended real-corpus total of **106 incomplete** omits the additional incomplete request from d047. The saved per-file outputs and answer breakdown show 106 incomplete for 7dce and one for d047, so the real total is **107**, alongside seven complete (114 requests total). The prose later states that correct 107 total. Thus a scope-aware review should credit correct fixture and per-file calculations while still flagging inconsistent top-level aggregation. This is a caller reporting issue, not an API reconstruction defect.

An additional source audit found a caller projection mistake outside the scored fields. C's `queryrepo/native/native-scan.js` reads `c.boundary.logicalParentUuid` and other guessed compaction properties instead of the documented `c.logicalParentUuid`, `c.metadata`, `c.beforeMessage`, `c.summaries`, and `c.boundary.source`. Its `out/inspect-synth.json` consequently shows null projected metadata even while listing the actual available keys. The first-following-message coordinate is correctly projected. The nulls should not be cited as missing native evidence: the wrapper discarded accessible fields. This limits the run's compaction-coordinate demonstration despite correct request counts and real-file hashes.

Finally, B's narrative incorrectly states that JavaScript `.length` for `abé` is four; it is three UTF-16 code units, while UTF-8 length is four bytes. The computed `unicode_bytes:4` is correct, but that explanatory sentence is not. B also claims that `queryOne` cannot distinguish empty results from broken queries immediately after documenting that broken queries throw; its own observed behavior establishes the distinction. These prose errors illustrate why correct structured answers cannot certify the entire debrief.

No fixed task text, oracle value, or raw grade was changed after inspecting these outcomes. A future experiment should require named `archive_source_ordinals` and `chronological_source_ordinals` fields, plus separate `synthetic_request_counts` and `real_request_counts` objects. This is a harness improvement for a subsequent cohort, not a retroactive change to this matrix.

## Evaluator correction v2: corpus-relative paths

C GLM/low/r1 exposed an evaluator bug. Its `real_scans` identifies inputs with unambiguous `framework/basename` paths relative to the supplied native corpus root. The prompt never required absolute paths. Evaluator v1 resolved those strings against its own working directory, causing an incorrect **3/42** score even though the emitted hashes, byte counts, record counts, and scopes matched the actual inputs.

Evaluator v2 accepts either an exact known absolute path or an exact, uniquely matching `framework/basename` path. It does not accept arbitrary cwd-relative paths, bare basenames, or ambiguous aliases. Duplicate canonical paths still fail row cardinality or identity coverage. Focused checks passed for valid relative paths and rejection of duplicate absolute/relative aliases and bare basenames. This changes path interpretation only; hashes, bytes, counts, scope, fixture oracle facts, and fixed prompts remain unchanged.

The original 3/42 result and its available individual checks are preserved in `sources/persona-adjudications.json` under `harness_path_resolution_bug_corrected_v2`. New summary receipts identify schema `gmt015-persona-independent-grades/v2` and explicitly list this correction. The corrected C GLM/low result is **40/42**: the remaining two raw mismatches are the previously identified count-scope ambiguity. Its top-level 7/107 correctly counts the real corpus, and its report contains the correct synthetic 2/1. Unlike GLM/high, the real incomplete aggregate here is internally correct. Do not classify the original path failures as model or product errors.
