#!/usr/bin/env python3
"""Verify immutable inputs/product and observed Pi session model/thinking metadata."""
import argparse,hashlib,json,pathlib
p=argparse.ArgumentParser(description=__doc__);p.add_argument('--repo',type=pathlib.Path,required=True);p.add_argument('--matrix',type=pathlib.Path,default=pathlib.Path('/private/tmp/gmt015-persona/matrix'));p.add_argument('--output',type=pathlib.Path,required=True);a=p.parse_args()
t=next(a.repo.glob('ttmp/2026/11/17/GMT-015*'));checks=[]
def digest(path):return hashlib.sha256(path.read_bytes()).hexdigest()
for name,expected in json.loads((t/'sources/persona-tested-source.json').read_text())['source_sha256'].items():
 checks.append({'kind':'product','path':name,'passed':digest(a.repo/name)==expected})
manifest=json.loads((t/'sources/persona-corpus-manifest.json').read_text())
for row in manifest['rows']:
 for key,hashkey in [('native_path','native_sha256'),('archive_path','archive_sha256')]:
  checks.append({'kind':'corpus','path':row[key],'passed':digest(pathlib.Path(row[key]))==row[hashkey]})
oracle=json.loads(pathlib.Path('/private/tmp/gmt015-persona/fixtures/oracle.json').read_text())
for name,expected in oracle['fixture_hashes'].items():
 checks.append({'kind':'fixture','path':name,'passed':digest(pathlib.Path('/private/tmp/gmt015-persona/inputs')/name)==expected})
settings=[]
for rfile in sorted(a.matrix.glob('*/receipt.json')):
 r=json.loads(rfile.read_text());models=[];thinking=[]
 for f in (rfile.parent/'sessions').glob('*.jsonl'):
  for line in f.read_text(errors='replace').splitlines():
   try:e=json.loads(line)
   except ValueError:continue
   if e.get('type')=='model_change':models.append({'provider':e.get('provider'),'model':e.get('modelId')})
   elif e.get('type')=='thinking_level_change':thinking.append(e.get('thinkingLevel'))
 settings.append({'run':r['run_id'],'requested_model':r['model'],'requested_thinking':r['requested_thinking'],'observed_models':models,'observed_thinking':thinking,'passed':bool(models and thinking) and all(m=={'provider':'lunaroute','model':r['model']} for m in models) and all(v==r['requested_thinking'] for v in thinking)})
result={'scope':'local file hashes and Pi session settings, not backend wire behavior or OS sandbox proof','input_checks':len(checks),'input_failures':[c for c in checks if not c['passed']],'settings':settings,'passed':all(c['passed'] for c in checks) and all(s['passed'] for s in settings)}
a.output.write_text(json.dumps(result,indent=2)+'\n');print(json.dumps({'passed':result['passed'],'input_checks':len(checks),'runs':len(settings),'settings_failures':sum(not s['passed'] for s in settings)}))
