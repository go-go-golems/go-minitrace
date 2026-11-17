#!/usr/bin/env python3
"""Summarize cohort receipts and diagnostic observations without publishing transcript bodies."""
import argparse,collections,json
from pathlib import Path
from common import load,save,verify
p=argparse.ArgumentParser();p.add_argument('--bundle',type=Path,required=True);p.add_argument('--cohort',required=True);p.add_argument('--output',type=Path,required=True);a=p.parse_args();config=verify(a.bundle)
grades={r['run_id']:r for r in load(a.bundle/'control'/f'{a.cohort}-grades.json')};runs=[]
for file in sorted((a.bundle/'runs'/a.cohort).glob('*/receipt.json')):
 r=load(file);g=grades.get(r['run_id'],{});statuses=collections.Counter();observations=collections.Counter()
 for cmd in (file.parent/'receipts').glob('*.json'):
  try:c=load(cmd)
  except ValueError:continue
  statuses[str(c.get('exit_code'))]+=1
  for stream in ['stdout_path','stderr_path']:
   path=Path(c.get(stream,''))
   if not path.is_absolute():path=Path(c['cwd'])/path
   if not path.is_file():continue
   body=path.read_text(errors='replace').lower()
   for marker in ['unknown flag','no such column','syntax error','incomplete','unsupported','queryresult','bad.js','operation not permitted']:
    if marker in body:observations[marker]+=1
 runs.append({k:r.get(k) for k in ['run_id','task','model','requested_thinking','exit_code','elapsed_seconds','tool_calls','tool_error_events','reported_usage','early_checkpoint','answer_present','timed_out','checkpoint_failed']}|{'checks_passed':g.get('passed',0),'checks_total':g.get('total',0),'mismatches':[c for c in g.get('checks',[]) if not c['passed']],'answer_error':g.get('answer_error'),'wrapped_exit_counts':dict(statuses),'diagnostic_observations':dict(observations),'sandbox_probe_count':len(r.get('sandbox_probes',[]))})
result={'schema_version':2,'cohort':a.cohort,'revision':config['revision'],'binary_sha256':config['binary_sha256'],'runs':runs,'totals':{'runs':len(runs),'answers':sum(bool(r['answer_present']) for r in runs),'exit_zero':sum(r['exit_code']==0 for r in runs),'early_checkpoints':sum(r['early_checkpoint'] is not None for r in runs),'tool_calls':sum(r['tool_calls'] or 0 for r in runs),'checks_passed':sum(r['checks_passed'] for r in runs),'checks_total':sum(r['checks_total'] for r in runs)},'limits':'Diagnostic string observations are not bug counts; expected negative probes also contain errors. Requested thinking is not backend verification. Review scripts and traces for causal claims.'}
save(a.output,result);print(json.dumps(result['totals']))
