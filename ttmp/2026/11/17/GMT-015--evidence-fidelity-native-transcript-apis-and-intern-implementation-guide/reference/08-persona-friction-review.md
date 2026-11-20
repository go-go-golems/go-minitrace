---
Title: Persona experiment friction review and independent command replay
Ticket: GMT-015
Status: active
Topics: [analysis, minitrace]
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Evidence-backed review of pilot and task A/D command attempts, with independently replayed D artifacts.
LastUpdated: 2026-11-17T16:00:00-05:00
WhatFor: Distinguish model mistakes, onboarding friction, and product defects without trusting model self-assessments.
WhenToUse: Review the persona experiment or prioritize API and documentation improvements.
---

# Final independent D replay coverage

All eight D matrix runs finished before their final JS artifacts were reviewed and independently replayed. Each private receipt pins reviewed source hashes and replay-script hash/configuration. Seven checks cover help, required-source rejection, an extra reviewer-selected unmatched source probe, three real framework/label invocations, and exact nine-session actual-tool-count parity. These are bounded checks, not an assertion that every task requirement passed. All eight preserved the intentionally invalid sibling and emitted its signed-default diagnostic; all eight showed required/runtime flags and rejected missing source.

| Model / effort / repetition | Passed checks | Actual sessions | Actual tools | Material outcome |
| --- | ---: | ---: | ---: | --- |
| DeepSeek / high / 1 | 7/7 | 9 | 706 | Exact identity and count parity |
| DeepSeek / high / 2 | 7/7 | 9 | 706 | Exact identity and count parity |
| DeepSeek / low / 1 | 7/7 | 9 | 706 | Exact identity and count parity |
| DeepSeek / low / 2 | 7/7 | 9 | 706 | Exact identity and count parity |
| GLM / high / 1 | 6/7 | 8 | 706 | WHERE/default filter drops zero-tool session |
| GLM / high / 2 | 6/7 | 8 | 706 | Recipe-derived ID selection drops zero-tool session |
| GLM / low / 1 | 6/7 | 8 | 706 | Inner join drops zero-tool session |
| GLM / low / 2 | 5/7 | 8 | 706 | Inner join omission plus unmatched-source crash |

The four-versus-four pattern describes these correlated task runs only. Do not extrapolate a model population error rate or an independent persona effect. In every failed parity case the aggregate706 still matches, demonstrating why per-session identity checks are required. Scripts and receipts retain output-shape differences rather than mistaking nested objects for incorrect results.

# Scope and method

This is a bounded qualitative audit, not a pass-rate estimate. The completed GLM low pilot was reviewed through its 31 tool executions; task A traces were inspected for specific command failures and recoveries. Task D generated JS repositories were reviewed before independent execution. The table below distinguishes replayed checks from full task completion. No new model calls or product changes were part of this review.

Evidence is in `/private/tmp/gmt015-persona/pilot/events.jsonl` and `/private/tmp/gmt015-persona/matrix/<run>/events.jsonl`. Tool-call numbers below count `tool_execution_end` events, not assistant turns. Shell output containing an error is not necessarily marked `isError`: several commands print an exit status and subsequently execute `cat`, leaving the shell process successful. Consequently the runner's `tool_error_events` is a transport-level measure, not a count of failed product commands. Native transcript errors returned as analytical data must also not be counted as failures of the reviewing agent.

Only API expressions, diagnostics, and aggregate results are quoted here. Raw historical transcript content remains in the private corpus. Model report claims are hypotheses until supported by tool output or independent replay.

## Pilot: successful UTF-8 demonstration after four failed product executions

The pilot's first four attempts to run its generated command failed; its fifth succeeded. It performed additional discovery and source inspection between attempts. These are separate from the initial provider-registration launch failure described below.

1. **Call 18, builder method guessed on the wrong receiver.** The generated expression was `mt.db().Content(JSON.stringify(ARCHIVE)).Name("demo.minitrace.json").SQLiteMemory().MaxCellBytes(3).Build()`. Running `go run ./cmd/go-minitrace query commands demo utf8-demo --query-repository /private/tmp/gmt015-persona/pilot/query-commands --output json` produced `TypeError: Object has no member 'Name'`. Recovery used `mt.sources().Content(...).Name(...).Build()` followed by `mt.db().Sources(sources)`. Classification: model misuse, with a discoverability opportunity. A fluent API can appear to permit all source modifiers on every receiver; short receiver-specific examples are more useful than suggesting a product regression.
2. **Calls 20 and 23, invalid hand-written normalized archive.** The synthetic object contained `metrics: []`. The loader reported `GoError: demo.minitrace.json is not native minitrace JSON and AutoConvert is disabled`. The agent first renamed the source to `utf8-demo-0001.minitrace.json`; this produced the same error. After reading the Go schema, it replaced the array with a metrics object. Classification: invalid caller input plus weak validation diagnostics. The message points toward format/conversion policy without identifying the incompatible field. The filename change was an ineffective model retry, not a required API convention.
3. **Call 30, raw SQL passed to recipe execution.** The expression `db.execute("SELECT content AS text FROM turns WHERE turn_index = 0")` failed with `TypeError: recipe.sql must be a string or sql() method`. This is consistent with the documented contract: execute accepts a recipe or `{sql,args}`, whereas queryResult accepts SQL text. The model had already retrieved the evidence-first API reference, so this is not evidence that the contract was absent. Recovery replaced execute with queryResult. Classification: model misuse; the diagnostic could explicitly suggest `queryResult(sql, ...args)` for a string argument.
4. **Call 31, successful final command.** The result contained `utf8ByteLength_abe: 4`. The permissive query returned text `ab`, `complete:false`, and one cell diagnostic with originalBytes 4 and emittedBytes 2. Strict mode returned `rows:[]`, `count:0`, and `error:"strict evidence: query result is incomplete"`. This supports the narrow UTF-8/strict-mode demonstration. It does not establish completion of the larger matrix task bank.

A preceding fixture-discovery command searched `testdata` for `*.minitrace.json`, then attempted `head` on the empty filename. Its `No such file or directory` result is shell/model handling of an empty search, not a go-minitrace failure.

## Task A, DeepSeek high repetition 1: SQL safety errors with useful recovery

At call 12 the agent attempted `SELECT name, type FROM sqlite_master WHERE type IN ('table','view') ORDER BY name`. The product rejected it with `query references disallowed table/view "sqlite_master"; use db.schema() or db.tables() from JS to introspect the schema`. This is an intentional policy boundary with an actionable diagnostic. It exposes onboarding friction for a SQL-first reader because introspection requires switching to JS; it is not evidence that schema introspection is broken.

At call 19 it supplied a SQL file containing multiple statements. The result was `Error: multiple SQL statements are not allowed`. The agent split its work into `counts`, `top-tools`, `range`, and `failures` SQL files and ran them separately at call 21. The counts invocation reported exit 0 and actual normalized child-row counts. Classification: model assumption about batch SQL, successfully corrected. Preserve the one-statement rule in the short onboarding path and show a JS command composing several queryResult calls.

Later raw evidence inspection encountered a historical command with a nonzero process exit. That content belongs to the source transcript and is not a failure of the current agent's inspection command. This distinction matters when automatically mining error strings.

## Task D, DeepSeek high repetition 1: intentional catalog error

The generated `commands/bad.js` defines metadata `limit: {type:"int", default:-1, ...}`. Catalog discovery reported `unsupported metadata literal "unary_expression"` within the field/default diagnostic, while continuing to expose valid sibling commands. This is the known signed-negative-default limitation in the pinned scanner, not accidental malformed JavaScript. It is deliberately requested by task D. A successful isolation test does not mean negative defaults are supported.

The generated `good.js` binds the required `source` input as a SQL parameter, uses a standard ToolRows recipe, and calls db.tables()/db.schema(). Its final source selects `agent_framework = ?`; therefore a reviewer-supplied `--source probe` is expected to yield zero selected sessions. That smoke test alone cannot validate tool counts. Independent replay must additionally use actual framework values and compare counts against the supplied archives.

# Harness and provider interpretation

The first pilot launch failed with `Unknown provider lunaroute` because `--no-extensions` also disabled provider registration. The harness repaired this with an explicit extension exception: `--no-extensions -e /Users/manuel.odendahl/.pi/agent/npm/node_modules/@lunaroute/pi-extension/src/index.ts`. This is a harness configuration error, not go-minitrace or task-model failure. Fresh matrix sessions retain this exception, along with isolation from context files, skills, and prompt templates.

Root verified local provider/model metadata: both chosen models use the openai-responses API, advertise supportsReasoningEffort, and map low to low and high to high. These are distinct locally requested settings. They do not prove that the remote backend honors the requested effort. Some other GLM levels collapse to the same advertised value; no inference about those levels is needed for this experiment. Usage reasoning fields can be recorded without turning them into a backend contract guarantee.

The provider extension hardcodes model cost coefficients to zero. Consequently zero reported usage.cost is not evidence of free billing. The matrix also pairs each persona with one task, so comparisons describe task/persona pairs, not an independently randomized persona effect.

# Review priorities

- Improve invalid normalized-input diagnostics to expose the failing field or parser reason before suggesting conversion. The pilot spent a retry changing a filename that could not repair its metrics shape.
- For execute(string), add a targeted receiver/signature hint in a future change. Preserve recipe semantics; this review does not propose overloading every query method.
- Keep a minimal runnable source-builder example beside the DB example. Emphasize which modifiers belong to sources versus db.
- Document the SQL introspection and one-statement boundaries in the first SQL-to-JS example.
- Treat signed metadata defaults as an outstanding scanner limitation; preserve sibling isolation tests separately.
- Count attempted commands with recorded exit values rather than relying exclusively on Pi tool error booleans or keyword matches.

# Independent replay log

## D-deepseek-4.1-flash-high-r1

This run finished with exit 0 and an answer file before independent replay. All four generated JS files were reviewed for scope before execution. The reviewer ran only `go run ./cmd/go-minitrace query commands good` with the explicit generated repository; no generated shell scripts were executed. The intentionally invalid bad.js stayed in place.

Five fresh invocations passed: ordinary --help, --source probe, and --source claude-code/codex/pi with the nine-archive corpus glob. Every invocation emitted the signed-default sibling diagnostic and the valid command remained usable. Help showed --source and --archive-glob. Probe returned an empty JSON array, as its framework filter implies. Each real-framework invocation returned a JSON array of three session rows. The combined nine per-session tool counts independently matched `len(tool_calls)` from each normalized archive: 706 total. This validates those counts, not historical causal ordering or every requirement in task D. Fresh CLI output was an ordinary array of row objects in this case; an array-return wrapping issue must not be asserted for this particular root-verb shape without a failing reproduction.

The private run contains `independent-replay.json` plus stdout/stderr captures. The receipt fingerprints all four reviewed JS files and records seven explicit checks (six invocations, including required-input rejection, and aggregate per-session parity). Its status is `pass_for_reviewed_checks`; this is deliberately narrower than a full task grade. Source review confirms bound parameters and recipe use. Separate cache-receipt and raw-return contrast assertions are not independently graded by these seven checks.

The matrix coverage table below supersedes this first replay checkpoint. Generated answer booleans are not independent grades.

## Additional corroboration: discovery and validation

The first A run's tool output confirms `discover claude-code --help` explicitly says subagent transcripts are ignored at this layer. The mismatch between a native discovery count and a supplied archive set containing a subagent is therefore a documented scope difference, not automatically lost data. The generic discover help still describes Claude Code and Codex in its introductory sentence while listing Copilot and Pi commands below; this is a small documentation inconsistency rather than a missing Pi implementation.

The same run executed `validate --path <its-converted-directory> --archive --output json` for Claude, Codex, and Pi. Each emitted `[]`. Its initial pipeline recorded the final pipeline command's exit, which alone would be weak validation evidence. The agent then reran each validation with direct stdout/stderr redirection and immediately inspected `$?`: all three returned 0, three-byte stdout, and empty stderr. This retry establishes successful validation of those three outputs. It also illustrates why checking a shell pipeline's apparent success is insufficient without pipefail or direct exit capture. An optional positive summary could improve onboarding; changing JSON findings output would need compatibility consideration.

## Additional A-run friction: flag scope and command markers

A-deepseek-4.1-flash-low-r1 and A-glm-5.3-flash-high-r1 both attempted `convert claude-code ... --run-record <file>` (and analogous conversion commands), receiving `Error: unknown flag: --run-record`. Both subsequently produced conversions. This is a repeated assumption that receipt flags apply uniformly across conversion adapters: Codex conversion does support --run-record; the observed unknown-flag failures concern Claude/Pi conversion. Query run also supports receipts. Classification: caller misuse with a cross-command discoverability opportunity. The task asked for command receipts but did not require this particular flag.

The DeepSeek low run also added `--cache-mode memory` to a generated `query commands session-counts` invocation and received `Error: unknown flag: --cache-mode`. The implemented cache flags belong to `query run`; JS commands configure their database builder. This is another scope assumption, not a failure of query-run cache controls.

The GLM high run initially generated `__export__({ sessionCounts });`; invoking its command produced `Error: ReferenceError: __export__ is not defined`. It replaced that marker with `__verb__("sessionCounts", {name:"session-counts", short:...})`. This is a guessed marker, corrected using the actual command-authoring convention. It supports a smaller copyable first-command tutorial rather than expanding the runtime to accept invented markers. The run finished with answer artifacts, but this review does not independently certify its full task completion from that fact.

## Reusable replay procedure

`scripts/persona-testing/replay_d.py` retains the fresh replay logic. Review every JS file in a completed run's commands directory first, then write `reviewed-js.json` mapping relative paths to SHA-256 hashes. The script refuses any added, removed, or changed JS file and any run lacking a completion exit_code. Invoke it with --repo, --run, --corpus, and --reviewed-hashes; --verb supports an explicitly reviewed command path. It never executes generated shell scripts. These checks are scope controls, not an OS security sandbox.

The checker captures ordinary help, omitted required --source, probe, and three framework selections; compares flat session_id/tool_count rows against actual archive tool arrays; and records JSON receipts beside private stdout/stderr. Different command semantics or output columns require manual grading. For the first D run, omission of --source was independently rejected with nonzero exit and a required-field diagnostic. Source inspection confirms SQL positional binding, and ToolRows result rows actually drive the aggregate counts.

## D-glm-5.3-flash-high-r1: successful execution misses a zero-tool session

After completion, all three generated JS files were reviewed and hash-pinned. Six independent invocation checks passed: help, missing-source rejection, probe, and three framework selections. The seven-check receipt nevertheless records partial_or_failed_checks because the combined per-session result contains eight sessions instead of nine. Its total remains 706 tool calls, exactly matching the full corpus total. Aggregate-total parity alone would falsely certify this result.

The generated query uses a LEFT JOIN but then adds `AND t.tool_call_id IS NOT NULL` in WHERE, eliminating sessions with no matching tools. It also introduces a min_tools parameter defaulting to 1 despite the task requesting per-session counts without that filter. The omitted Codex session has zero tools. This is a model-authored analytical query defect, not an incorrect count returned by the database. The product correctly executes the query it was given. Setting min_tools to zero would not repair the WHERE exclusion.

Source inspection confirms source and min_tools use positional SQL parameters. The standard ToolRows recipe executes and its count appears as normalized_tool_rows; it does not drive the per-session count calculation. Its error is checked, but complete is not checked. This review observed a small corpus and did not trigger a row-limit omission in that recipe, so failure to check complete is a latent evidence-handling risk rather than an observed truncation defect here. The root good verb returns flat JSON row arrays in this run as well.

## D-deepseek-4.1-flash-low-r1: independently verified actual counts

All six generated JS files were reviewed after completion. Seven replay checks passed, including required-source rejection and all nine actual per-session counts. This command names its actual count column actual_tool_calls and separately exposes declared_tool_calls from metadata. The replay script was extended with explicit --count-column selection so it cannot accidentally validate a declared aggregate instead. Source inspection confirms the source filter is bound as a LIKE argument; interpolation constructs the parameter value rather than SQL text. A separate tool-rows command executes the standard recipe and exposes its envelope. Main per-session counts use independent bound SQL. This is composition across commands, not a missing main-command recipe dependency.

## D-glm-5.3-flash-low-r1: nested output and another zero-tool omission

Two final JS files were reviewed after completion. Six invocation checks pass, but per-session parity fails: again eight of nine sessions and the same 706 tools. The query starts from tool_calls and INNER JOINs sessions, so zero-tool sessions cannot appear. Its nested return is legitimate: the CLI emits one object containing a sessions array. Replay therefore uses explicit --rows-key sessions; output shape is not the reason for the parity failure.

The code constructs `mt.query().ToolRows().Build()` and reads its SQL and args, but executes a different custom COUNT query through db.execute. The comment describes a standard ToolRows recipe demonstration more strongly than the implementation supports. This is an unmet execution requirement, not a failing recipe API. The task also triggered `GoError: no such column: agent_framework` before a join repair, and `unknown flag: --cache-mode` when flags belonging to query run were applied to query commands. Both are caller errors. No source edits were made by the reviewer.

## Confirmed declaration defect: nullable error-envelope arrays

`pkg/minitracejs/typescript.go:34` declares QueryResult.columns as string[] and rows as Array<Record<string, unknown>>. `pkg/minitracedb/query.go:116` and multiple validation/query error paths return QueryResult{Error:...} without initializing those slices. JSON represents those nil slices as null; matrix observations of bad-SQL queryResult rows:null are consistent with the implementation. This is a product type/runtime contract mismatch, separate from the already documented distinction between throwing convenience methods and error envelopes. Priority: fix before recommending blind typed iteration over error results. A future patch should either normalize columns/rows to empty arrays on every error path or declare their nullability and update examples. Do not change this contract mid-experiment.

CLI permissive query output is rows-oriented, so a displayed clipped value alone does not expose its diagnostics. Existing --run-record receipts and --strict-evidence are concrete mitigations; do not characterize the CLI as unable to preserve completeness evidence. Teaching analysts to inspect the receipt or select strict mode is part of the current usable path.

## Helper inference hypothesis rejected by controlled reproduction

An A repetition-2 trace showed unknown --archive-glob after factoring DB creation into `_withDB(mt, fn)`. This initially suggested static runtime-flag inference might miss helper calls. A controlled reviewer-authored reproduction compared an inline builder with an equivalent helper and ran both help and actual query paths. Both expose --archive-glob and return nine sessions. A second control put the handlers in separate files to avoid same-file inference contamination; both still pass. These are negative results against the hypothesized product bug.

Inspection of the actual failed command explains it: `counts/session-counts.js` contains two verbs, session-counts and top-tools. The agent invoked `query commands counts session-counts` and `counts top-tools`; the former is a group, the latter skips the file-stem group. The correct leaf paths include `counts session-counts session-counts` or `counts session-counts top-tools`. This is nested CLI path misuse. It motivates clearer command-path examples/help diagnostics, not an inference-engine patch. Repros and four-command receipt are preserved in scripts/persona-testing/repros/runtime-archives-helper.js, runtime-archives-inline.js, and receipt-separate.json.

## D repetition-2 observations

DeepSeek high repetition 2 independently passes seven checks, including all nine actual per-session counts selected from tool_rows_from_recipe. It executes ToolRows, groups its actual rows, and keeps sessions with zero matching tools. Completeness is surfaced in output rather than enforced; the observed query was complete.

GLM high repetition 2 passes six invocation checks but misses the zero-tool session. It obtains sessionIds from Object.keys(recipeCounts), then uses a bound SQL IN list over those IDs. Thus it excludes the session before projecting counts, despite checking ToolRows completeness correctly. Its source flag is an output label, not a framework filter; this is compatible with the task wording. Replays collect the union of per-session results and do not wrongly require the label to filter. This third independently reproduced omission reinforces why identity-set parity matters alongside totals, but the correlated small experiment cannot estimate general model failure probabilities.

# Categorized conclusions and action priorities

| Category | Verified finding | Practical improvement |
| --- | --- | --- |
| Prompt / evaluation | Successful D commands can silently omit zero-tool sessions while matching total706 | Explicitly require all supplied session identities, including zero rows; grade sets and per-session counts |
| Harness | Provider isolation disabled provider registration; tool error booleans omit shell-level failures | Keep explicit provider extension; preserve direct exit codes and raw receipts; distinguish expected negative probes |
| Documentation | Agents repeatedly transfer query-run flags to convert/query-commands and guess registration markers | Add one minimal runnable command, command-family flag map (including adapter-specific conversion differences), and multi-verb path example |
| API ergonomics | Invalid normalized object produces generic format/conversion error; execute(string) gives recipe-shape error | Preserve underlying parse reason; offer targeted queryResult hint for string input |
| Confirmed product defect | QueryResult declaration says arrays while early error envelopes emit null | Normalize arrays or type nullability consistently; add negative-path contract test |
| Known product limitation | Signed numeric metadata default -1 is rejected, but valid siblings remain usable | Fix pinned scanner separately while retaining isolation tests |
| Agent analytical errors | JOIN/WHERE or recipe-derived ID sets remove zero-tool sessions | Start from sessions and left join/aggregate tool rows; maintain identity parity tests |
| Rejected hypothesis | Helper-wrapped RuntimeArchives loses flags | No product fix: controlled repro passes; actual issue was nested CLI path misuse |

The strongest conclusions are narrow and reproducible. Model/tool success is not an analytical grade. A new prompt should clarify invariants without embedding fixture answers: every selected session survives aggregation, zero is not missing, labels and filters have explicit semantics, and counts come from actual rows. A future evaluation should retain the original prompt as a baseline and compare a separately fingerprinted revision; do not retroactively repair agents' original answers.

## Final repetition-2 replay details

DeepSeek low repetition2 retains zero-tool sessions by starting from sessions and supplying an empty aggregate where no recipe tools exist. After diagnosing incomplete ToolRows evidence it raises MaxCellBytes to2000000 and enforces error/complete before counting. Seven independent checks pass. Its bound SQL excludes empty framework strings; all nine supplied archives have known frameworks, so this corpus does not test unknown-framework preservation.

GLM low repetition2 independently fails the unmatched source probe with `TypeError: Cannot read property 'session_id' of undefined`: it dereferences sessions[0] before handling an empty selection. Actual framework invocations succeed. The arbitrary probe value is an additional reviewer edge case, not one of the original valid framework inputs; its crash does not imply failure of the requested valid-command demonstrations. The per-session count query begins with tool_calls INNER JOIN sessions and omits the zero-tool session, independently confirmed as8 rather than9 sessions. It also appends a textual diagnostic row whose session_id begins RECIPE_TOOLROWS and whose tool_calls field contains an error description. The reviewer explicitly selected the numeric tool_calls count field and excluded this textual pseudo-row; the omission is not a parsing artifact. This is a generated-command empty-input/aggregation defect, not a runtime failure on valid DB data. Recommended prompt and example changes should keep diagnostics separate from analytical rows and handle empty selections before indexing.

The final replay harness records its own SHA-256, command args, explicit count-column/nested-row configuration, reviewed JS hashes, direct exit codes, and evidence paths. Earlier receipts were rerun under the hash-recording harness; the final case needed an additional explicit numeric-value guard for its diagnostic pseudo-row. Distinct script hashes are preserved rather than retroactively claiming identical evaluator code.
