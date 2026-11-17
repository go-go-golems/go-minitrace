---
Title: Persona-based agent testing guide for transcript evidence workflows
Ticket: GMT-015
Status: active
Topics: [analysis, minitrace]
DocType: design-doc
Intent: long-term
Summary: Planned controlled Pi agent matrix, broad task bank, reproducible grading, and honest interpretation of model and thinking comparisons.
---

# Persona-based agent testing guide

## 1. Purpose and current status

This experiment tests whether an agent can use go-minitrace to answer realistic transcript questions, discover the right APIs, preserve evidence, and explain limitations. Passing Go unit tests establishes implementation behavior under fixtures. It does not establish that a new user or coding agent can find the relevant command, understand its result, or avoid a plausible but incorrect analytical conclusion. The persona matrix exercises that additional boundary.

The planned matrix contains **32 runs**: four task groups, two models, two requested thinking settings, and two repetitions. Four personas are paired with the task groups. Personas are **not** a fifth independently crossed factor. The experiment therefore compares particular persona/task workflows; it cannot isolate an independent causal effect of persona instructions.

The runner has launched the 32-run matrix after a successful GLM/low pilot. Nine frozen real sources contain 793 normalized turns and 706 tool records, with recorded activity from September 15 through November 12, 2026. The pilot executed byte-length, clipping and strict-rejection checks, correcting several API mistakes along the way. Matrix results are recorded separately in reference/07 and sources/persona-matrix-results.json. Requested model settings and observed provider behavior remain different fields; no free-billing or backend-effort guarantee is inferred from the CLI.

The implementation under test is the evidence-fidelity worktree described in [the intern architecture guide](01-trustworthy-transcript-analysis-architecture-implementation-and-intern-guide.md). Pin its Git revision and binary/source build before the pilot. Do not change the implementation, documentation or task prompts halfway through a comparison and combine the resulting runs as if the inputs were identical.

## 2. Matrix and persona/task pairing

| Task group | Paired persona | Main work | Main failure to detect |
|---|---|---|---|
| A — Corpus inventory | New intern | Discover CLI surfaces, identify sources, convert/validate selected records, report a defensible inventory | Wrong denominator, guessed CLI flags, counting copies as independent sessions |
| B — Fidelity and provenance | Python analyst | Replace hand traversal with SQL/JS; inspect byte limits, arguments, ordering, statuses and token origins | Silent evidence loss, false success, mixing estimates with reported usage |
| C — Native forensics | Forensic reviewer | Stream native records, inspect reference completeness and compaction metadata, distinguish physical from causal order | Treating incomplete requests as zero, inventing replay certainty, leaking future results |
| D — Command authoring and catalog | SQL analyst acting as command author | Package a reusable parameterized command; discover required help, recover from invalid siblings and document errors | SQL interpolation, unreadable help, whole-catalog failure, incorrect envelope handling |

Use the same persona and task prompt for both models and both thinking settings within a group. The model IDs selected from the current Pi catalog are `glm-5.3-flash` and `deepseek-4.1-flash`, both through provider `lunaroute`. The requested thinking values are `low` and `high`.

```text
4 fixed persona/task pairs
        x 2 model IDs
        x 2 requested thinking levels
        x 2 repetitions
        = 32 planned matrix slots
```

A complete four-persona by four-task cross would instead contain 128 runs with these model, thinking and repetition factors. That is not the current plan. Likewise, two repetitions do not justify a strong statistical claim about a model's general quality. They help reveal obvious instability and prevent one lucky execution from being the only evidence.

A persona specifies a working style and background, not privileged answers. The intern should receive plain-language project orientation, the Python analyst a request to compare familiar JSON traversal with existing APIs, the forensic reviewer an emphasis on provenance and uncertainty, and the SQL analyst an emphasis on reusable query interfaces. Do not give one persona the answer key or extra API documentation unless that difference is an explicit experiment variable.

## 3. Inputs, isolation and provenance

Use two complementary input collections. The real corpus contains nine selected sources with a frozen manifest. Its purpose is to expose realistic record shapes, sizes, provider conventions and analytical ambiguity. Nine sources do not automatically mean nine unique sessions or nine independent user tasks. The manifest and normalized identities establish the actual denominator.

Synthetic fixtures supply known answers for cases that real data may not exercise reliably: a multibyte clipping boundary, an unknown tool status, identical and conflicting archive copies, a missing native blob, a forward-only reference, an invalid catalog sibling, and a tool emitted before a proposed cutoff whose result arrives later. Never treat a synthetic expected answer as a measurement from the real corpus.

For each run, retain:

- Implementation revision, build/version identity, Pi version, provider/model ID and requested thinking setting.
- Persona/task prompt hashes, fixture/manifest hashes, run ID, repetition and ordering.
- Launch directory, enabled tool list, explicit context files and extension-discovery settings.
- Start/end timestamps, wall limit, exit status, timeout/transport/parse status and retry relationships.
- Pi JSON output, tool-call transcript, generated artifacts, final answer and bounded reflection.
- Reported usage/cost fields with their provenance; missing values remain unknown.
- Grader version, machine-check results and reviewer findings.

Give each attempt its own working/output directory. Fixture originals remain read-only by policy and are hash-checked before and after; the agent writes derived archives, SQL, JS and reports only under its assigned output directory. A command-line tool allowlist is not a filesystem security sandbox: the `bash` tool can still run programs with its process permissions. The runner and environment should enforce any required isolation rather than claiming that a prompt creates it.

Use exact source lists or a manifest where possible. A broad glob that includes previous run outputs can change the denominator on the second repetition. Keep output trees outside source selection, record conversion diagnostics and preserve all original logs. No credentials or unrelated personal settings belong in the task packet or published artifacts.

## 4. Pi invocation and the pilot gate

Pi 0.87.1 help verifies the flags below. The parent runner will use these supported controls:

```bash
pi \
  --provider lunaroute \
  --model glm-5.3-flash \
  --thinking low \
  --mode json \
  --print \
  --session-dir /absolute/path/to/run/session \
  --no-context-files \
  --no-skills \
  --no-extensions \
  -e /Users/manuel.odendahl/.pi/agent/npm/node_modules/@lunaroute/pi-extension/src/index.ts \
  --no-prompt-templates \
  --tools read,bash,write,edit \
  --append-system-prompt /absolute/path/to/persona-and-boundaries.txt \
  @/absolute/path/to/task-prompt.md
```

Replace the model and thinking value for the corresponding slot. Paths are illustrative run paths, not preexisting files. The explicit `-e` path is the installed lunaroute provider extension on this machine. An initial pilot launch with only `--no-extensions` failed with `Unknown provider lunaroute`: disabling discovery also removed provider registration. Keep discovery disabled but load this one declared provider extension, pinning its version/source hash in the run configuration. On another machine, locate the installed equivalent rather than assuming this absolute path exists.

The task prompt must supply the necessary orientation explicitly because automatic context and skill discovery are disabled. This keeps global or project-specific guidance from silently changing one run's input.

`--session-dir` controls session storage; it is not the source corpus directory. `--mode json` makes output available for structured capture, but the grader must still distinguish transport output from the model's final answer. Do not assume every stdout line is the final JSON response or that a single successful process exit proves the analytical task passed.

The planned operational limits are approximately eight minutes per run and at most four concurrent runs. The external runner owns these limits; the example does not invent a Pi timeout flag. On timeout, preserve partial logs and identify the slot as incomplete. Do not silently give slow models extra time or exclude their timeouts from the denominator.

Before the full batch, pilot the execution path and requested thinking settings. At minimum establish that both model IDs resolve through the intended provider, tools execute, session/log output is captured, artifacts can be graded, and cleanup preserves originals. A pilot should also exercise low/high settings enough to detect immediate rejection or normalization of unsupported values.

The catalog's claim that a model supports thinking is not proof that the upstream API honored both requested levels. Record the requested setting, any model/adapter-resolved setting, provider metadata and warnings separately. If honoring the distinction cannot be verified, label the comparison “requested low versus requested high through this Pi/provider configuration.” Reasoning-token differences alone do not prove that a particular provider control was applied.

Pilot runs count toward the 32 slots only if their final prompts, fixtures, constraints, implementation and grading configuration exactly match those slots. If a pilot changes the runner or prompt, retain it as pilot evidence and restart the affected matrix slots. Do not erase unsuccessful pilots or count retries as independent repetitions.

No billing claim is established by the catalog, provider name or a zero/missing receipt field. Report actual recorded cost only when its source is known. Otherwise state that cost was unavailable or unverified.

## 5. Twenty-question task bank

The bank below covers broad usage. The four matrix prompts should select coherent bundles from it rather than asking every agent to complete all 20 questions within eight minutes. Task prompt versions must specify exactly which questions are graded. Freeze that selection before the batch.

### A — Corpus inventory and entry-level CLI use

**Q01 — What commands exist for this source?** Starting from CLI help, identify the installed discover, convert, validate and query surfaces relevant to the supplied corpus. Save the help/version evidence and the commands actually used. Grade correct discovery, not memorization of command names from another version. A guessed unsupported flag followed by successful recovery is friction; confidently reporting that it ran is a correctness failure.

**Q02 — Which sources belong in the requested inventory?** Use supplied paths, cwd/activity metadata and the task's selection rule to produce a source manifest. Distinguish source files, normalized sessions, parents/children where available, and independent tasks. Grade selected identities against the frozen answer key. A cwd match is a filter, not proof that implementation work occurred in that repository.

**Q03 — Can selected native sources be converted and validated reproducibly?** Inspect the relevant subcommand's help, convert only the selected files into a derived directory, and run the archive validator. Retain diagnostics and before/after source hashes. Grade conversion success, identity preservation and honest reporting of unsupported/malformed inputs. Never require the agent to mutate originals to make validation pass.

**Q04 — What is the defensible corpus summary?** Return counts of selected sources, unique normalized IDs, turns and tools with a reproducible SQL or JS artifact. Include earliest/latest recorded timestamps with their meaning. Grade equality to frozen totals and null handling; an empty or missing field is not automatically zero. Require the final answer to name the denominator it uses.

### B — Fidelity, SQL/JS projection and provenance

**Q05 — How can the schema be inspected?** Find tables/columns through supported help or `db.schema()`/`db.tables()`. Explain why querying SQLite's internal catalog can be blocked by the read-only allowlist. Grade whether the agent discovers the supported route rather than weakening authorization or inventing a nonexistent table.

**Q06 — Can an exact session query remain parameterized?** Implement a bound SQL query and a JS recipe execution using a supplied session identifier containing SQL-shaped punctuation. Return only the exact intended match. Grade the saved code as well as output: string concatenation that happens to work on a benign identifier is not acceptable. `db.execute({sql, args})` and standard recipe objects are valid routes.

**Q07 — Does the agent detect UTF-8 evidence clipping?** Query `abé` with a three-byte limit. Explain why the valid result prefix is `ab`, originalBytes 4 and emittedBytes 2, with complete false. Repeat in strict evidence mode and verify that no partial rows are returned. Grade metadata and byte/code-point distinction. The pre-fix replacement-character output must not be accepted as intact evidence.

**Q08 — Can it distinguish archive order and original status?** Extract tools in `session_id, source_ordinal` order, retaining `arguments_json`, nullable `success`, original `output_status` and derived `outcome_status`. Use same-turn tools whose IDs sort differently from archive order. Grade exact tool order and preservation of unknown status. `source_ordinal` is not a native line number or a result-arrival time.

**Q09 — Can it keep usage origins separate?** Compare known-zero usage, missing accounting and tool-output estimates. Explain `token_origin`, `token_unit`, `token_method` and `reported_usage_fields`. Grade that estimates are not merged into provider-reported prompt totals and that presentation views are not double-counted. A numerical sum without its measurement origin is insufficient.

**Q10 — Can it resolve a corpus conflict explicitly?** Run `mt.corpusManifest(paths)` on an identical-copy pair and divergent archives sharing an ID. Identify duplicate index zero correctly, explain byte equivalence versus semantic equivalence, and propose a declared selection policy. Grade detection and preservation of variants. The helper does not automatically enforce a winner during legacy ingestion.

### C — Native forensics and attribution

**Q11 — Can it stream records and interpret hash scope?** Use `mt.native().ForEach` to count record kinds while collecting malformed-record diagnostics. Compare an EOF scan with an intentional callback stop. Grade line/byte coordinates, whole-source versus consumed-prefix hashing, and unchanged source bytes. A prefix digest presented as the whole-file digest is a critical error.

**Q12 — What exactly is missing from a request?** Inspect fixtures with complete references, missing shape, missing base, missing blob and an unsupported tail form. Explain each status and retain all diagnostics. Grade against structural expectations. “Complete” means the supported reference graph resolved locally, not that the provider's full request schema or execution was certified.

**Q13 — What can a compaction boundary establish?** Inspect a fixture whose logical parent appears physically after the boundary. Describe boundary/summary/preserved-message relationships and quote source coordinates. Grade that physical file order is not called causal conversation order. A file attachment should not be counted as a fresh Read action merely because it resembles a tool result.

**Q14 — Is a proposed turn cutoff a faithful historical snapshot?** Review a tool emitted before the cutoff whose result arrives afterward. Explain why an emitting-turn filter alone can include future result knowledge. Grade explicit acknowledgment that result-arrival provenance/replay certification is not implemented. The correct answer may be “unsupported for this causal claim,” not a clever-looking incomplete query.

**Q15 — Which session actually performed the work?** Start from a supplied file/commit or implementation claim, use supported history/candidate queries, and inspect the decisive tool action plus external repository evidence when supplied. Distinguish planning, a write attempt, successful execution and a verified resulting commit. Grade a cited action/identity chain, not frequency of path mentions or the model's retrospective confidence.

### D — Reusable commands, failure recovery and reflection

**Q16 — Can it author a reusable repository-backed query command?** Create a small JS command with a required session/source parameter, a bounded output projection and bound SQL. Invoke it through the actual catalog and save its output. Grade reproducibility and argument semantics, not an unexecuted code block. Verify both normal help and actual execution.

**Q17 — Does one malformed sibling disable useful commands?** Add a controlled invalid command next to a valid one in the assigned scratch catalog. Show the diagnostic and run the valid sibling. Also test a negative numeric metadata default. Grade the distinction: catalog isolation is implemented, but the pinned signed-literal parser limitation remains. Do not claim the invalid command itself became supported.

**Q18 — Can it select an appropriate cache policy?** Run a read-only query in an environment where a normal home-cache write is undesirable, choosing documented memory/none behavior or an explicit cache directory. Capture the chosen settings. Grade that the agent understands materialization/cache writes are separate from SQL table mutation and does not request unnecessary database write permissions.

**Q19 — Can it explain and handle error shapes?** Compare invalid builder/recipe input, a SQL execution error, and strict evidence rejection. Show which paths throw and which return an envelope with `error`. Grade that the command cannot emit clipped rows as successful analysis merely because no JavaScript exception occurred. Existing contracts remain deliberately mixed; the task is to use them correctly.

**Q20 — What did the agent find difficult, and is that account supported?** Request a short post-task reflection naming confusing APIs/help, failed attempts, workarounds, unnecessary context and suggested improvements. Require concrete command or artifact references and separate “observed” from “I think.” Grade the reflection against the actual tool trace. Reflection is qualitative evidence for usability, not an authority that overrides objective task checks.

## 6. Prompt contract and expected artifacts

Every task prompt should identify the assigned role, the analytical question, allowed source locations, output directory, required artifacts and stopping condition. It should explicitly ask the agent to inspect current help rather than assume a familiar command exists. Keep the same help and implementation available across model/settings cells.

A compact task contract can follow this pattern:

```text
Role: Python analyst migrating an existing evidence workflow.
Question: [fixed task-group questions and inputs]
Inputs: [frozen manifest and fixture paths]
Write only: [this run's output directory]
Required outputs:
  - runnable SQL/JS or command script
  - machine-readable answers with units and denominators
  - short report with evidence coordinates and limitations
  - bounded reflection citing actual failed/successful commands
Do not modify source logs or implementation.
If unsupported, preserve the exact error and explain the limitation.
```

The grader should not require a particular implementation when several supported routes are correct. A SQL analyst can use direct SQL; a Python analyst can use Python for independent comparison where the task permits it. If the question specifically tests the JS API, require an actually executed JS artifact for that part. Otherwise a solution that bypasses the API entirely cannot establish its usability.

Avoid grading prose elegance as a substitute for correctness. A short answer with the correct denominator and a runnable query is stronger evidence than a polished narrative with invented counts. Conversely, a correct number alone is insufficient if the saved code cannot reproduce it or silently relies on unintended input files.

## 7. Grading and critical failures

Use deterministic checks for fixture facts and human review for analytical interpretation. Preserve per-question results rather than collapsing everything into a single attractive score. Suggested dimensions are completion, factual correctness, reproducibility, evidence fidelity, discovery friction and quality of limitations.

A suggested per-question rubric is:

- **Pass:** correct result, supported reasoning, executable evidence and required provenance.
- **Partial:** meaningful correct work with a bounded omission or an explicitly unresolved limitation.
- **Fail:** incorrect result, unsupported certainty, missing required execution or a critical evidence error.
- **Not attempted:** no relevant work before the stopping condition.
- **Infrastructure blocked:** provider/runner failure prevented a valid task attempt; retain separately from analytical failure.

Critical failures should remain visible even if many easy checks pass. Examples include modifying original sources, inventing a successful command, accepting clipped evidence as complete, turning unknown bytes into zero, claiming a causal replay without arrival data, or publishing unrelated sensitive content. A model with such a failure should not be described simply as “90% correct.”

Machine checks should parse the artifacts and compare identities, counts, byte lengths, null values, ordering and status enums. They should also verify that the required command actually ran successfully. Human review checks whether the interpretation follows from those artifacts and whether an exact phrase match is being mistaken for causation.

Freeze the grader before the full matrix where practical. If a defect in the grader is found, retain its original result, record the correction, and rerun grading consistently across all affected slots. Do not change the acceptance rule only for a preferred model. Display-only normalization may be appropriate for harmless formatting differences, but it must not create evidence or silently turn a failed model answer into a different substantive answer.

## 8. Metrics and comparisons

Report planned slots, started attempts, completed transports, parsed answers, graded outputs and successful task outcomes separately. Retries increase attempt count, not the number of independent repetitions. A partial output after timeout can be assessed qualitatively, but it is not a completed run.

For each task/model/requested-thinking combination, show both repetitions. Useful descriptive metrics include time to first successful query, total wall time, tool-call count, failed-command count, successful recovery count, graded question results, artifact reproducibility and critical-error flags. Report token/cost measurements only with their source and uncertainty.

A low/high comparison is conditional on the recorded Pi/provider configuration. If actual reasoning-setting support remains unverified, say so prominently. Do not infer that high thinking is ineffective because a provider ignored the setting, or that one model is faster because its failed calls returned immediately. Concurrency can affect latency, so retain launch order and overlap information.

Model reflections should be summarized alongside trace evidence. For example, “required parameter hidden in help” can be checked against the saved help output; “I did not need the instructions” is a subjective judgment that cannot establish absence of influence. Count repeated usability complaints as candidates for investigation, not automatic product requirements.

The final report should lead with task-specific outcomes and concrete failure modes. With two repetitions per cell, avoid significance claims, global rankings or broad statements about all coding work. This is an engineering acceptance and usability study of a controlled task bank, not a general intelligence benchmark.

## 9. Execution sequence and handoff

First freeze the implementation, corpus/fixture manifests, prompts, persona mapping, grader and runner configuration. Validate the fixture expectations without model calls. Run a pilot that exercises provider selection, requested thinking, tool execution, transcript capture, timeout handling and grading. Fix infrastructure problems before scaling out, preserving failed pilot evidence.

Next launch the 32 planned slots with at most four concurrent runs and unique output directories. Record starts and outcomes incrementally so a process failure does not erase the entire batch. Resume only missing or explicitly retryable slots with matching configuration hashes. Do not reuse a prior success solely because its task name matches after changing the prompt or fixtures.

Then grade all completed artifacts and review critical failures. Investigate disagreements between automatic checks, final answers and model reflections. Publish a matrix receipt, per-task examples, concrete recommendations and a clear account of missing/blocked slots. Link the saved commands so a colleague can reproduce representative successes and failures without rerunning all models.

The parent runner owns actual model execution and outcome receipts; this guide does not perform either. At handoff, update the following status fields from those receipts:

| Status field | Current documentary state |
|---|---|
| Matrix slots | 32 planned |
| Real-corpus sources | 9 frozen selected sources; 793 turns and 706 tool records |
| Model IDs | Selected from Pi catalog; GLM/low launch in progress, broader resolution pending |
| Requested thinking |low/high; provider honoring not yet verified |
| Pilot completion | GLM/low pilot completed actual JS checks after explicit provider-extension loading; excluded from the 32 matrix slots |
| Full-matrix completion |Pending recorded runner outcomes |
| Billing |Not verified as free; use reported cost with provenance when available |

## 10. References and authoring notes

- [Intern architecture and implementation guide](01-trustworthy-transcript-analysis-architecture-implementation-and-intern-guide.md): exact API contracts, implemented scope and remaining work.
- [Query fidelity notes](../reference/02-query-fidelity-notes.md): UTF-8 and strict-evidence probes.
- [Provenance notes](../reference/03-provenance-notes.md): archive ordinal, usage labels and corpus conflicts.
- [Native evidence notes](../reference/04-native-evidence-notes.md): scan hashes, reference completeness and compaction limits.
- Ticket scripts and generated run/grade receipts: authoritative source for the actual matrix configuration and outcomes as they are produced.

The guide's author checked the installed Pi help without invoking a model. The help command attempted to acquire the normal global settings lock and reported a sandbox permission warning; help still printed. That warning is not a provider-readiness or billing test. No private settings content was inspected, and no model calls were made while writing this guide.

Keep the task bank broader than the initial matrix so later product changes can select focused regression tasks. Expand the experiment only when the additional questions answer an actual uncertainty. The goal is concrete evidence about whether agents can use these APIs correctly, with enough provenance that another engineer can retrace the result.
