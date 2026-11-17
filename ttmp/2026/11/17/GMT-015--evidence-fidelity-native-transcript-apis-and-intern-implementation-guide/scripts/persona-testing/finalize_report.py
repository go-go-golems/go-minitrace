#!/usr/bin/env python3
"""Reconcile completed matrix metrics into the guide and review; no model execution."""
import collections,json,pathlib,re,statistics
ROOT=pathlib.Path(__file__).resolve().parents[7]
# Resolve by the script's actual ticket, independent of cwd.
T=pathlib.Path(__file__).resolve().parents[2]
d=json.loads((T/'sources/persona-matrix-results.json').read_text());runs=d['runs']
if d['finished_runs']!=32 or len(runs)!=32:raise SystemExit('requires all 32 final receipts')
rows=[]
for model,level in sorted({(r['model'],r['requested_thinking'])for r in runs}):
 group=[r for r in runs if (r['model'],r['requested_thinking'])==(model,level)]
 rows.append(f"| {model} | {level} | {len(group)} | {statistics.median(r['elapsed_seconds'] for r in group):.1f} | {sum(r['tool_calls'] for r in group)} | {sum(r.get('correct_fields',0)for r in group)}/{sum(r.get('graded_fields',0)for r in group)} |")
table='\n'.join(['| Model | Requested thinking | Finished runs | Median seconds | Tool calls | Literal A/B/C checks |','|---|---|---:|---:|---:|---:|',*rows])
metric=f'''## Completed experiment results

All **32 matrix runs** finished with exit code zero and produced both answer.json and report.md. There were **zero timeouts**. They emitted **1,671 tool calls** against a frozen product implementation. The separate successful GLM/low pilot is excluded from these totals, as is its earlier failed provider-registration startup.

The real corpus contains nine historical sessions: three Claude Code, three Codex and three Pi, with 793 normalized turns and 706 tool records. Recorded activity spans September 15–November 12, 2026. Inputs were selected for framework/size/shape diversity, not randomly sampled.

{table}

The literal evaluator reports **550/560 A/B/C fields**. These are exact answer-contract checks, not a percentage of tasks successfully completed. Ten mismatches concern underspecified dataset/order fields: agents sometimes supplied real or combined request counts instead of fixture counts, or chronological/real-archive ordinals instead of fixture archive order. Supporting computations and separate substantive errors are adjudicated in sources/persona-adjudications.json. No model ranking is justified by this tiny, fixed-order, paired-persona study. Wall times include tool execution and overlapping Go processes; requested thinking is recorded in Pi metadata, not independently verified backend reasoning behavior.

All **eight D commands** were source-reviewed and independently replayed. All four DeepSeek-generated variants preserve all nine session identities; all four GLM-generated variants omit the zero-tool session despite keeping the correct aggregate total of 706. Every variant enforces its required source flag and allows a valid command beside the malformed signed-default sibling. An additional reviewer probe using an unmatched source exposes one GLM-generated empty-result crash; valid framework invocations still succeed. This extra edge probe is not conflated with the original requested task.

All 32 persisted Pi sessions were converted and queried through go-minitrace itself. Native, normalized and runner tool/failure accounting and the five compared usage fields match, with no reported warnings. The source/input integrity check passes all 324 hash checks and all 32 model/thinking metadata checks. This verifies unchanged known inputs and product files; it does not prove workspace-scope compliance.

That distinction matters: two agents created 12 untracked log/output files in the repository through relative shell redirects. The files were preserved under /private/tmp/gmt015-persona/scope-artifacts and removed from the product worktree by moving them, not deleting evidence. The manifest is sources/persona-scope-violations.json. The next harness should use a hash-pinned binary from each run directory and enforce a read-only repository, rather than depending on prompts alone.

The ticket preserves 127 matrix-generated source artifacts plus the pilot script, the task runner, independent graders, reviewed replay controls, corpus provenance and aggregate receipts. Raw transcripts and streamed model outputs remain private. A durable local compressed evidence archive is recorded in sources/persona-private-evidence.json under ~/.local/share/go-minitrace/experiments/GMT-015; temporary-file cleanup will not erase the only retained copy. All-run event scanning and targeted deep review are distinguished explicitly; this is not a claim that every transcript line was manually reviewed.
'''
# Replace a generated metric section idempotently.
def insert_metrics(text):
 start='<!-- final-matrix-metrics:start -->';end='<!-- final-matrix-metrics:end -->'
 block=start+'\n'+metric+'\n'+end+'\n'
 if start in text:return text[:text.index(start)]+block+text[text.index(end)+len(end):].lstrip('\n')
 pos=text.find('\n## ')
 return text[:pos]+'\n\n'+block+text[pos:]
review=T/'reference/10-persona-testing-findings-and-improvement-plan.md'
s=review.read_text();s=re.sub(r'^Topics:.*(?:\n[ \t]+-.*)*','Topics: [analysis, minitrace]',s,flags=re.M)
s=s.replace('The matrix is still progressing when this note is first written.','All 32 runs and the eight independent command replays are complete.')
s=s.replace('three reviewed GLM-authored count queries','all four reviewed GLM-authored count queries')
s=s.replace('Product: CLI evidence visibility','API gap: CLI evidence visibility')
s=s.replace('so a safe existing recovery is to use strict mode or JS envelopes.','so existing recoveries are strict mode, JS envelopes, or a separate CLI `--run-record` receipt containing completeness and cell diagnostics. The CLI can preserve that evidence today; the gap is its absence from permissive row-only stdout.')
s=s.replace('Repeated A attempts use `convert --run-record` and receive `unknown flag`;','Repeated A attempts use `convert claude-code` or `convert pi` with `--run-record` and receive `unknown flag`; the Codex converter does support that flag, so this is a subcommand-specific distinction, not a blanket conversion limitation;')
s=s.replace('The other two B/C runs were still active and excluded from substantive outcome review.','That was an intermediate snapshot. The final all-run event audit covers all 32 finished runs; the snapshot count is preserved here only to describe the bounded deep-review pass.')
s=s.replace('root\'s final audit will refresh complete-matrix coverage.','the final complete-matrix audit is /private/tmp/gmt015-persona/trace-audit.json.')
s=s.replace('Report timeout denominators separately once every receipt is final, and retain incomplete outputs rather than retroactively omitting difficult attempts.','All 32 final receipts record zero timeouts. Keep that denominator separate from field correctness and preserve this rule for future cohorts.')
s=insert_metrics(s)
s=s.split('\n## Prompt and harness changes for the next cohort')[0]
s += '''
## Prompt and harness changes for the next cohort

Use a small explicit JSON schema whose names identify the dataset and order: synthetic.request_counts, real.request_counts, archive_source_ordinals and chronological_source_ordinals. State whether real totals include the fixture, name the accepted path base, and require an identity list alongside aggregates. These changes address demonstrated ambiguity; they must be evaluated in a new cohort, not retroactively inserted into this one's prompts.

Separate mandatory acceptance checks from optional exploration, and require intermediate artifacts before long debriefs. Two high-thinking runs finished at 450 and 473 seconds under the 480-second limit; a successful final answer does not mean the workflow is efficient. Record shell exit status directly instead of masking errors through echo/cat pipelines. Prebuild the tested binary once, record its hash and revision, and keep Go caches fixed. This also avoids the observed fresh-HOME dependency-cache side effect.

Enforce filesystem permissions for the repository and evaluator, not merely a prose prohibition. Preserve attempted violations and classify them independently from product correctness. A future documentation experiment can compare help-only discovery, a short task-oriented quickstart, and the full reference using the same inputs and explicit answer schema. That would measure which documentation helps, rather than asking models to retrospectively guess what influenced them.
'''
review.write_text(s)
guide=T/'design-doc/02-persona-based-agent-testing-guide.md';s=guide.read_text()
s=s.replace('Planned controlled Pi agent matrix','Executed controlled Pi agent matrix').replace('The planned matrix contains **32 runs**','The executed matrix contains **32 runs**')
s=s.replace('The runner has launched the 32-run matrix after a successful GLM/low pilot.','The runner completed all 32 runs after a successful GLM/low pilot.')
s=s.replace('| Matrix slots | 32 planned |','| Matrix slots | 32/32 finished; all answer and report artifacts present |').replace('| Model IDs | Selected from Pi catalog; GLM/low launch in progress, broader resolution pending |','| Model IDs | Both observed in all requested runs; source/model metadata verified |').replace('| Full-matrix completion |Pending recorded runner outcomes |','| Full-matrix completion | Complete: zero timeouts, 1,671 tool calls |')
s=s.replace('The parent runner owns actual model execution and outcome receipts; this guide does not perform either. At handoff, update the following status fields from those receipts:','The saved runner owns model execution; the following fields are reconciled from final receipts:')
s=re.sub(r"The guide's author checked the installed Pi help.*?No private settings content was inspected, and no model calls were made while writing this guide\.\n",'The initial provider-registration failure and successful explicit-extension recovery are preserved in the diary. All actual model runs are recorded by the saved runner; authoring-only helper activity is excluded from experiment counts.\n',s,flags=re.S)
s=s.split('\n## 11. Completed run results and review')[0]
s += '\n## 11. Completed run results and review\n\n'+metric.replace('## Completed experiment results\n\n','')+'\nSee [the detailed improvement review](../reference/10-persona-testing-findings-and-improvement-plan.md) for actual command failures, confirmed bugs, rejected hypotheses and acceptance criteria.\n'
guide.write_text(s)
p=T/'reference/07-persona-test-results.md';s=p.read_text();start=s.index('The first independent snapshot') if 'The first independent snapshot' in s else s.index('The final independent snapshot');end=s.index('\n\nThe matrix',start)
s=s[:start]+'The final independent snapshot contains **32 planned, 32 started and 32 finished runs**, with 32 answer/report pairs and zero timeouts. Literal A/B/C field checks are 550/560; the ten scope/order mismatches are annotated separately from substantive analytical errors. All eight D commands have reviewed replay receipts. The final machine-readable result is sources/persona-matrix-results.json (evaluator v2).'+s[end:];s=s.replace('Summary: Partial experiment status','Summary: Completed experiment status');p.write_text(s)
p=T/'reference/09-persona-session-self-analysis.md';s=p.read_text().replace('Root can rerun this command after additional matrix runs finish; the current result is a snapshot, not a claim that all planned runs have completed.','The final run covers all 32 finished sessions: all 32 normalize and validate, with zero tool-count mismatches, zero compared failure/usage mismatches and no warnings. Earlier checkpoints below are historical snapshots.');p.write_text(s)
# Public receipts include parameters/paths, not native transcript content.
receipts=[json.loads(p.read_text())for p in sorted(pathlib.Path('/private/tmp/gmt015-persona/matrix').glob('*/receipt.json'))]
(T/'sources/persona-run-receipts.json').write_text(json.dumps(receipts,indent=2)+'\n')
audit=json.loads(pathlib.Path('/private/tmp/gmt015-persona/trace-audit.json').read_text())
public=[{'run':r['run'],'tool_calls':r['tool_calls'],'explicit_tool_error_events':len(r['error_events']),'attempted_features':r['attempted_features'],'evaluator_access_candidates':len(r['potential_scope_violations'])}for r in audit['runs']]
(T/'sources/persona-trace-coverage.json').write_text(json.dumps({'scope':audit['scope'],'runs':public,'workspace_scope_violation_manifest':'persona-scope-violations.json'},indent=2)+'\n')
print(json.dumps({'runs':len(runs),'metrics_table_rows':len(rows),'review_words':len(review.read_text().split()),'guide_words':len(guide.read_text().split())}))
