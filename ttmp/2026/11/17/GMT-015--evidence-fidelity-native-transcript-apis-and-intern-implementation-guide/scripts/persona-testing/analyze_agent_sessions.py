#!/usr/bin/env python3
"""Independently normalize finished persona Pi sessions; compare SQL and runner units."""
import argparse,collections,hashlib,json,subprocess
from pathlib import Path

def digest(b):return hashlib.sha256(b).hexdigest()
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--matrix',type=Path,default=Path('/private/tmp/gmt015-persona/matrix'));p.add_argument('--out',type=Path,default=Path('/private/tmp/gmt015-persona/agent-analysis'));args=p.parse_args();args.out.mkdir(parents=True,exist_ok=True)
 ticket=Path(__file__).resolve().parents[2];runs=[];sources=[];commands=[];warnings=[]
 def command(argv,label):
  full=['go','run','./cmd/go-minitrace']+argv;result=subprocess.run(full,text=True,capture_output=True)
  for ext,value in [('stdout',result.stdout),('stderr',result.stderr)]: (args.out/(label+'.'+ext)).write_text(value)
  commands.append({'argv':full,'exit_code':result.returncode,'stdout_path':str(args.out/(label+'.stdout')),'stderr_path':str(args.out/(label+'.stderr'))})
  if result.returncode:raise RuntimeError(f'{label} failed; inspect {args.out}')
  return result.stdout
 for receipt in sorted(args.matrix.glob('*/receipt.json')):
  r=json.loads(receipt.read_text())
  if 'elapsed_seconds' not in r:continue
  run={'run_id':r['run_id'],'model':r.get('model'),'persona':r.get('persona'),'task':r.get('task'),'exit_code':r.get('exit_code'),'elapsed_seconds':r['elapsed_seconds'],'receipt_sha256':digest(receipt.read_bytes()),'runner_tool_start_events':r.get('tool_calls'),'runner_tool_error_events':r.get('tool_error_events'),'runner_reported_usage':r.get('reported_usage'),'sources':[]}
  for path in sorted((receipt.parent/'sessions').glob('*.jsonl')):
   body=path.read_bytes();sha=digest(body);dest=args.out/'raw'/run['run_id']/path.name;dest.parent.mkdir(parents=True,exist_ok=True)
   if dest.exists() and digest(dest.read_bytes())!=sha:raise ValueError(f'Previously finished source changed: {path}')
   dest.write_bytes(body);assert digest(path.read_bytes())==sha
   native_usage=collections.Counter();types=collections.Counter();calls=[];errors=0;results=0;sid=None
   for line in body.splitlines():
    if not line.strip():continue
    d=json.loads(line);types[d.get('type','unknown')]+=1
    if d.get('type')=='session':sid=d.get('id')
    msg=d.get('message',{});
    if msg.get('role')=='assistant':
     for key,value in (msg.get('usage') or {}).items():
      if isinstance(value,(int,float)) and not isinstance(value,bool):native_usage[key]+=value
     for b in msg.get('content',[]) if isinstance(msg.get('content'),list) else []:
      if b.get('type')=='toolCall':calls.append(b.get('name','unknown'))
    if msg.get('role')=='toolResult':results+=1;errors+=msg.get('isError') is True
   item={'session_id':sid,'original_path':str(path),'snapshot_path':str(dest),'sha256':sha,'bytes':len(body),'native_record_types':dict(types),'native_assistant_tool_calls':len(calls),'native_tool_names':dict(collections.Counter(calls)),'native_tool_result_events':results,'native_error_result_events':errors,'native_assistant_usage_sum':dict(native_usage)}
   run['sources'].append(item);sources.append(str(dest))
  if not run['sources']:warnings.append(f"{run['run_id']}: completed receipt has no persisted sessions")
  runs.append(run)
 sql=[]
 if sources:
  listing=args.out/'sources.txt';listing.write_text('\n'.join(sources)+'\n')
  command(['convert','pi','--source-list',str(listing),'--output-dir',str(args.out/'archives'),'--output','json'],'convert')
  command(['validate','--path',str(args.out/'archives'),'--archive','--output','json'],'validate')
  sql=json.loads(command(['query','run','--archive-glob',str(args.out/'archives/active/*/*.minitrace.json'),'--sql-file',str(Path(__file__).with_name('session-analysis.sql')),'--output','json'],'sql'))
 byid={r['session_id']:r for r in sql};mapping={'input':'recorded_input','output':'recorded_output','cacheRead':'recorded_cache_read','cacheWrite':'recorded_cache_write','reasoning':'recorded_reasoning'}
 for run in runs:
  native_tools=sum(x['native_assistant_tool_calls'] for x in run['sources']);native_errors=sum(x['native_error_result_events'] for x in run['sources']);normalized=[]
  for source in run['sources']:
   q=byid.get(source['session_id']);source['normalized_sql']=q
   if q is None:warnings.append(f"{run['run_id']}: missing normalized identity {source['session_id']}")
   else:normalized.append(q)
  run['comparison']={'native_assistant_tool_calls':native_tools,'normalized_tools':sum(q['normalized_tools'] for q in normalized),'native_error_result_events':native_errors,'normalized_failed_tools':sum(q['normalized_failed_tools'] for q in normalized),'normalized_unknown_tools':sum(q['normalized_unknown_tools'] for q in normalized)}
  c=run['comparison'];c['runner_vs_native_tool_delta']=None if run['runner_tool_start_events'] is None else c['native_assistant_tool_calls']-run['runner_tool_start_events'];c['normalized_vs_native_tool_delta']=c['normalized_tools']-native_tools;c['normalized_failure_vs_runner_error_delta']=None if run['runner_tool_error_events'] is None else c['normalized_failed_tools']-run['runner_tool_error_events']
  c['usage']={}
  for key,col in mapping.items():
   nv=sum(s['native_assistant_usage_sum'].get(key,0) for s in run['sources']);vals=[q[col] for q in normalized if q.get(col) is not None];rv=(run['runner_reported_usage'] or {}).get(key);sv=sum(vals) if vals else None
   c['usage'][key]={'native_assistant_sum':nv,'sql_recorded_sum':sv,'sql_reporting_sessions':len(vals),'runner_event_sum':rv,'sql_minus_native':None if sv is None else sv-nv,'runner_minus_native':None if rv is None else rv-nv}
 result={'schema':'gmt015-persona-session-analysis/v1','finished_receipt_count':len(runs),'native_session_count':len(sources),'normalized_session_count':len(sql),'runs':runs,'commands':commands,'warnings':warnings,'limits':['Finished means receipt has elapsed_seconds, including timeout/nonzero exits.','Runner counts tool-start/error events; native counts assistant toolCall and toolResult records; normalized SQL counts adapter tool rows. These are separate units.','Usage sums retained assistant records versus runner events; not independent billing or exact marginal cost.','Only completed persisted sessions are converted; outputs and transcripts remain private.','No source mutation, model invocation or historical command replay.']}
 out=ticket/'sources/persona-session-analysis.json';out.write_text(json.dumps(result,indent=2)+'\n');(args.out/'summary.json').write_text(json.dumps(result,indent=2)+'\n')
 print(json.dumps({'finished':len(runs),'native_sessions':len(sources),'normalized_sessions':len(sql),'tool_mismatches':sum(r['comparison']['normalized_vs_native_tool_delta']!=0 or r['comparison']['runner_vs_native_tool_delta']!=0 for r in runs),'warnings':warnings}))
if __name__=='__main__':main()
