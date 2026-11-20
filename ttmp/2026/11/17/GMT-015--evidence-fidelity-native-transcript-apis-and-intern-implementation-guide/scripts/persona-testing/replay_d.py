#!/usr/bin/env python3
"""Replay only human/agent-reviewed, hash-pinned task-D JavaScript repositories.
Review ALL .js files before writing reviewed-js.json {relative_path: sha256}.
No generated shell scripts are executed. This is not an OS sandbox.
"""
import argparse, hashlib, json, pathlib, subprocess

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--repo',type=pathlib.Path,required=True)
    p.add_argument('--run',type=pathlib.Path,required=True)
    p.add_argument('--corpus',type=pathlib.Path,required=True)
    p.add_argument('--reviewed-hashes',type=pathlib.Path,required=True)
    p.add_argument('--verb',nargs='+',default=['good'])
    p.add_argument('--rows-key',help='Reviewed nested row field inside a CLI object result')
    p.add_argument('--count-column',default='tool_count',help='Reviewed output column containing actual tool counts')
    a=p.parse_args(); commands=a.run/'commands'
    current={str(f.relative_to(commands)):hashlib.sha256(f.read_bytes()).hexdigest() for f in commands.rglob('*.js')}
    approved=json.loads(a.reviewed_hashes.read_text())
    if not current or current!=approved:raise SystemExit('Review gate: current JS files differ from explicit reviewed hashes')
    receipt=json.loads((a.run/'receipt.json').read_text())
    if 'exit_code' not in receipt:raise SystemExit('Run is still active; do not replay mutable artifacts')
    captures=a.run/'independent-replay';captures.mkdir(exist_ok=True)
    glob=str(a.corpus/'archives/*/active/*/*.minitrace.json')
    base=['go','run','./cmd/go-minitrace','query','commands',*a.verb,'--query-repository',str(commands)]
    checks=[]; actual={};count_rows=[]
    cases=[('help',['--help']),('missing-source',['--archive-glob',glob,'--output','json'])]
    cases += [(v,['--source',v,'--archive-glob',glob,'--output','json']) for v in ['probe','claude-code','codex','pi']]
    for label,flags in cases:
        cmd=base+flags
        try:r=subprocess.run(cmd,cwd=a.repo,text=True,capture_output=True,timeout=90)
        except subprocess.TimeoutExpired:
            checks.append({'id':label,'passed':False,'error':'reviewer command timeout','command':cmd});continue
        out=captures/(label+'.stdout');err=captures/(label+'.stderr');out.write_text(r.stdout);err.write_text(r.stderr)
        check={'id':label,'passed':r.returncode==0,'exit_code':r.returncode,'command':cmd,'stdout':str(out),'stderr':str(err),'bad_sibling_diagnosed':'unsupported metadata literal' in r.stderr}
        if label=='help':check['passed'] &= '--source' in r.stdout and '--archive-glob' in r.stdout
        elif label=='missing-source':check['passed']=r.returncode!=0 and 'required' in (r.stdout+r.stderr).lower()
        else:
            try:
                data=json.loads(r.stdout);check['output_shape']=type(data).__name__
                if a.rows_key:
                    containers=data if isinstance(data,list) else [data]
                    data=[row for container in containers if isinstance(container,dict) for row in container.get(a.rows_key,[])]
                if label!='probe' and isinstance(data,list):
                    for row in data:
                        if isinstance(row,dict) and 'session_id' in row and isinstance(row.get(a.count_column),(int,float)) and not isinstance(row.get(a.count_column),bool):
                            actual[row['session_id']]=row[a.count_column];count_rows.append(row)
            except ValueError:check['passed']=False;check['error']='stdout not JSON'
        checks.append(check)
    expected={}
    for f in (a.corpus/'archives').glob('*/active/*/*.minitrace.json'):
        data=json.loads(f.read_text());expected[data['id']]=len(data['tool_calls'])
    checks.append({'id':'all_nine_actual_tool_counts','passed':actual==expected,'actual_sessions':len(actual),'expected_sessions':len(expected),'actual_tool_total':sum(actual.values()),'expected_tool_total':sum(expected.values()),'evidence':'Independent normalized archive tool_calls array lengths, not aggregate metrics. Recognizes session_id and configured actual-count column only; unmatched output requires manual grading.'})
    result={'replay_script_sha256':hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest(),'replay_config':{'verb':a.verb,'count_column':a.count_column,'rows_key':a.rows_key},'task':'D','reviewer':'port_normalized_js','status':'pass_for_reviewed_checks' if all(x['passed'] for x in checks) else 'partial_or_failed_checks','source_sha256':current,'checks':checks,'limits':'Does not grade all task requirements; model report claims are separate. Count parser expects flat session_id and explicitly selected count column.'}
    (a.run/'independent-replay.json').write_text(json.dumps(result,indent=2))
    print(json.dumps({'run':a.run.name,'status':result['status'],'checks':[{k:x[k] for k in ('id','passed')} for x in checks]}))
if __name__=='__main__':main()
