#!/usr/bin/env python3
"""Grade V2 answers against private data and host replay; never execute worker shell text."""
import argparse,json
from pathlib import Path
from common import load,save,sha,verify

def same(a,b):
 if isinstance(a,bool) or isinstance(b,bool):return type(a) is type(b) and a==b
 return a==b

def grade(answer,task,rows,fixture,bundle,receipt):
 if not isinstance(answer,dict):raise ValueError('answer must be a JSON object')
 checks=[]
 def check(key,actual,expected):checks.append({'key':key,'actual':actual,'expected':expected,'passed':same(actual,expected)})
 check('schema_version',answer.get('schema_version'),2);check('task',answer.get('task'),task)
 expected=fixture['expected']
 if task=='A':
  got=answer.get('sessions',[]);got=got if isinstance(got,list) else [];by={r.get('session_id'):r for r in got if isinstance(r,dict)}
  check('session_ids',sorted(str(k) for k in by),sorted(r['session_id'] for r in rows));check('session_row_count',len(got),len(rows));check('sql_js_equal',answer.get('sql_js_equal'),True)
  for row in rows:
   for k in ['turn_count','tool_count']:check(row['session_id']+'.'+k,by.get(row['session_id'],{}).get(k),row[k])
 elif task=='B':
  got=answer.get('synthetic',{});got=got if isinstance(got,dict) else {}
  for k in ['unicode_bytes','clipped_text','clipped_complete','strict_rows','strict_error','manifest_duplicate_count','manifest_conflict_count']:check('synthetic.'+k,got.get(k),expected[k])
  check('synthetic.archive_source_ordinals',got.get('archive_source_ordinals'),[0,1,2]);check('synthetic.chronological_source_ordinals',got.get('chronological_source_ordinals'),[1,2,0])
  ids={r.get('session_id') for r in answer.get('real_sessions',[]) if isinstance(r,dict)};check('real_frameworks',sorted({r['framework'] for r in rows if r['session_id'] in ids}),['claude-code','codex','pi'])
 elif task=='C':
  check('path_base',answer.get('path_base'),str(bundle/'corpus/raw'));check('exact_replay_possible',answer.get('exact_replay_possible'),False)
  got=answer.get('synthetic_request_counts',{});got=got if isinstance(got,dict) else {}
  check('synthetic.complete',got.get('complete'),expected['native_complete_requests']);check('synthetic.incomplete',got.get('incomplete'),expected['native_incomplete_requests'])
  def pathmap(key):
   values=answer.get(key,[]);values=values if isinstance(values,list) else [];mapped={}
   for item in values:
    if not isinstance(item,dict):continue
    value=item.get('path');path=Path(value) if isinstance(value,str) else Path('__invalid__')
    if path.is_absolute():
     try:path=path.relative_to(bundle/'corpus/raw')
     except ValueError:pass
    mapped[str(path)]=item
   expectedpaths=sorted(str(Path(r['native_path']).relative_to(bundle/'corpus/raw')) for r in rows)
   check(key+'.paths',sorted(mapped),expectedpaths);check(key+'.row_count',len(values),len(rows));return mapped
  scans=pathmap('real_scans');counts=pathmap('real_request_counts')
  for row in rows:
   p=Path(row['native_path']);key=str(p.relative_to(bundle/'corpus/raw'));body=p.read_bytes();records=[json.loads(line) for line in body.split(b'\n') if line.strip()]
   exp={'sha256':sha(p),'bytes':len(body),'lines':len(body.split(b'\n'))-(1 if body.endswith(b'\n') else 0),'hashScope':'whole_source'}
   for k,v in exp.items():check(key+'.'+k,scans.get(key,{}).get(k),v)
   check(key+'.api_request_records',counts.get(key,{}).get('api_request_records'),sum(r.get('type')=='api-request' for r in records))
 elif task=='D':
  replay=receipt.get('host_replay') or {};commands=replay.get('commands',[]);queries=commands[:3]
  # Explicit == 0: never `code or 1`, which incorrectly turns a successful zero into failure.
  passed=len(queries)==3 and all(r.get('exit_code')==0 and not r.get('timed_out') for r in queries)
  check('host.good_command_succeeded',passed,True);check('claimed.good_command_succeeded',answer.get('good_command_succeeded'),passed)
  helptext=Path(commands[3]['stdout_path']).read_text() if len(commands)==4 else ''
  check('host.required_source_visible','--source' in helptext,True);check('host.runtime_flags_visible','--archive-glob' in helptext,True)
  stderr='\n'.join(Path(r['stderr_path']).read_text() for r in commands)
  check('host.bad_sibling_diagnosed','bad.js' in stderr,True)
  check('claimed.negative_default_supported',answer.get('negative_default_supported'),False)
  by={r['session_id']:r for r in rows}
  for i,r in enumerate(queries):
   try:
    value=load(r['stdout_path']);value=value[0] if isinstance(value,list) and len(value)==1 else value
    if not isinstance(value,dict):raise ValueError('expected one JSON row')
    expectedrow=by[replay['selected_session_ids'][i]] # replay records use selected source order below
    check(f'host.query{i}.session_id',value.get('session_id'),expectedrow['session_id'])
    check(f'host.query{i}.tool_count',value.get('tool_count'),expectedrow['tool_count'])
   except (ValueError,KeyError,IndexError):check(f'host.query{i}.valid_json_count',False,True)
 return checks

def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--bundle',type=Path,required=True);p.add_argument('--cohort',required=True);a=p.parse_args();a.bundle=a.bundle.resolve();verify(a.bundle);rows=load(a.bundle/'private-oracle/corpus.json')['rows'];fixture=load(a.bundle/'private-oracle/fixtures.json');results=[]
 for path in sorted((a.bundle/'runs'/a.cohort).glob('*/receipt.json')):
  receipt=load(path)
  if 'elapsed_seconds' not in receipt:continue
  try:answer=load(path.parent/'answer.json');checks=grade(answer,receipt['task'],rows,fixture,a.bundle,receipt);error=None
  except (OSError,ValueError,TypeError) as e:checks=[];error=str(e)
  results.append({'run_id':receipt['run_id'],'answer_error':error,'checks':checks,'passed':sum(c['passed'] for c in checks),'total':len(checks),'early_checkpoint_present':receipt.get('early_checkpoint') is not None,'exit_code':receipt.get('exit_code')})
 save(a.bundle/'control'/f'{a.cohort}-grades.json',results);print(json.dumps(results,indent=2))
if __name__=='__main__':main()
