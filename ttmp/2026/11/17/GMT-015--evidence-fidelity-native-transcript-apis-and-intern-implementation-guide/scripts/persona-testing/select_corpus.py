#!/usr/bin/env python3
"""Freeze nine explicitly selected historical sources, convert, validate and inventory.
No models run. Raw data stays in /private/tmp. Run from the go-minitrace worktree.
"""
import argparse,collections,hashlib,json,subprocess
from pathlib import Path

SELECTED={
'claude-code':[
'.claude/projects/-Users-manuel-odendahl-code-tulip-playback/4db0e19b-4e9e-4b6a-9979-9036c370eb7c/subagents/agent-ae7076749749daca0.jsonl',
'.claude/projects/-Users-manuel-odendahl-code-tulip-playback/d047a697-35c0-460c-8fda-541799b5b5b0.jsonl',
'.claude/projects/-Users-manuel-odendahl-code-tulip-experiments-2026-09-14--factory-videos/7dce992c-7793-4e1a-950c-4db92cef2502.jsonl'],
'codex':[
'.codex/sessions/2026/10/13/rollout-2026-10-13T13-24-04-01a1364a-f58a-7ce3-8460-a856dacc8a1a.jsonl',
'.codex/sessions/2026/10/12/rollout-2026-10-12T14-20-10-01a13157-f631-7cd3-8f84-e2666b24af01.jsonl',
'.codex/sessions/2026/11/10/rollout-2026-11-10T11-59-20-01a1c666-5f02-7b92-984e-99a27afd7a94.jsonl'],
'pi':[
'.pi/agent/sessions/--Users-manuel.odendahl-code-tulip-playback--/2026-09-29T17-08-34-793Z_01a0ee23-bd68-75c1-a81f-2c43771fc216.jsonl',
'.pi/agent/sessions/--Users-manuel.odendahl-code-tulip-playback--/2026-09-17T17-19-05-437Z_01a0b061-0cdd-7557-a80a-96802d3d02c5.jsonl',
'.pi/agent/sessions/--Users-manuel.odendahl-code-tulip-tulip--/2026-09-15T15-50-28-655Z_01a0a5c3-342f-7447-be74-b6c35f542ff7.jsonl']}
def sha(b):return hashlib.sha256(b).hexdigest()
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--root',type=Path,default=Path('/private/tmp/gmt015-persona/corpus'));p.add_argument('--freeze-only',action='store_true');a=p.parse_args();a.root.mkdir(parents=True,exist_ok=True)
 ticket=Path(__file__).resolve().parents[2];rows=[];runs=[]
 def run(args,label):
  cmd=['go','run','./cmd/go-minitrace']+args;r=subprocess.run(cmd,capture_output=True,text=True)
  (a.root/(label+'.stdout')).write_text(r.stdout);(a.root/(label+'.stderr')).write_text(r.stderr)
  runs.append({'argv':cmd,'exit_code':r.returncode,'stdout':str(a.root/(label+'.stdout')),'stderr':str(a.root/(label+'.stderr'))})
  if r.returncode:raise RuntimeError(f'{label} failed; inspect private receipt')
  return r.stdout
 for framework,paths in SELECTED.items():
  copies=[]
  for rel in paths:
   source=Path.home()/rel;body=source.read_bytes();assert len(body)<20_000_000
   native_sha=sha(body);dest=a.root/'raw'/framework/source.name;dest.parent.mkdir(parents=True,exist_ok=True)
   if dest.exists() and sha(dest.read_bytes())!=native_sha:raise ValueError(f'Refusing changed snapshot: {dest}')
   dest.write_bytes(body);assert sha(source.read_bytes())==native_sha,'source changed during copy';copies.append(str(dest))
   events=[json.loads(line) for line in body.splitlines() if line.strip()];times=[r.get('timestamp') for r in events if isinstance(r.get('timestamp'),str)]
   row={'framework':framework,'source_path':str(source),'native_path':str(dest),'native_sha256':native_sha,'native_bytes':len(body),'native_records':len(events),'native_type_counts':dict(collections.Counter(r.get('type','unknown') for r in events)),'first_native_timestamp':min(times) if times else None,'last_native_timestamp':max(times) if times else None};rows.append(row)
  source_list=a.root/(framework+'-sources.txt');source_list.write_text('\n'.join(copies)+'\n')
  if a.freeze_only:continue
  run(['discover',framework,'--source-dir',str(a.root/'raw'/framework),'--output','json'],'discover-'+framework)
  out=a.root/'archives'/framework
  args=['convert',framework,'--source-list',str(source_list),'--output-dir',str(out),'--output','json']
  if framework=='codex':args+=['--run-record',str(a.root/'codex-conversion-run.json')]
  run(args,'convert-'+framework)
  run(['validate','--path',str(out),'--archive','--output','json'],'validate-'+framework)
  archives=list(out.rglob('*.minitrace.json'));assert len(archives)==3,(framework,len(archives))
  for arc in archives:
   b=arc.read_bytes();d=json.loads(b);prov=d.get('provenance',{});native=prov.get('source_path','');matches=[r for r in rows if r['framework']==framework and (r['native_path']==native or Path(r['native_path']).name==Path(native).name)]
   if len(matches)!=1:raise ValueError(f'Cannot map archive provenance {arc}: {native}')
   r=matches[0];tools=d.get('tool_calls',[]);turns=d.get('turns',[]);names=collections.Counter(t.get('tool_name') for t in tools);r.update(archive_path=str(arc),archive_sha256=sha(b),session_id=d['id'],source_format=prov.get('source_format'),cwd=d.get('environment',{}).get('working_directory'),timing=d.get('timing'),turn_count=len(turns),tool_count=len(tools),tool_names=dict(names),tool_success_counts=dict(collections.Counter(str(t.get('output',{}).get('success')) for t in tools)),file_target_count=sum(len(t.get('input',{}).get('file_targets') or []) for t in tools),truncated_tool_outputs=sum(bool(t.get('output',{}).get('truncated')) for t in tools))
 baseline=None
 if not a.freeze_only:
  sqlpath=Path(__file__).with_name('baseline.sql')
  output=run(['query','run','--archive-glob',str(a.root/'archives/*/active/*/*.minitrace.json'),'--sql-file',str(sqlpath),'--output','json'],'sql-baseline')
  baseline=json.loads(output);byid={r['session_id']:r for r in baseline};assert len(byid)==9
  for row in rows:
   query=byid[row['session_id']]
   assert query['turn_count']==row['turn_count'] and query['tool_count']==row['tool_count']
   assert query['unknown_tool_success_count']==row['tool_success_counts'].get('None',0)
   if row['tool_count']:assert query['min_source_ordinal']==0 and query['max_source_ordinal']==row['tool_count']-1
 manifest={'schema':'gmt015-persona-corpus/v1','selection':'Explicit historical sources nearest small/medium/large size strata; Claude includes one child and two parents; no current experiment logs. Not a representative random sample.','rows':rows,'sql_baseline':baseline,'sql_parity':'PASS' if baseline is not None else 'not run','commands':runs,'limits':['Native and normalized counts are different units; normalization may merge/split source records.','Tool success null is unknown; estimates and counts do not establish task correctness.','Sources selected by size/activity metadata, not claims of implementation authorship.','Raw sources and normalized bodies are private scratch only.']}
 (a.root/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n');(ticket/'sources').mkdir(exist_ok=True);(ticket/'sources/persona-corpus-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
 print(json.dumps({'sources':len(rows),'normalized':sum('archive_path' in r for r in rows),'root':str(a.root)}))
if __name__=='__main__':main()
