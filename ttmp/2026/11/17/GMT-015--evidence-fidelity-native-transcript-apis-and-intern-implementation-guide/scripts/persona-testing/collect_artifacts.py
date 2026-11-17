#!/usr/bin/env python3
"""Preserve model-authored JS/SQL/shell/Python, excluding transcripts and caches.
Review copied files before committing; this is copying, not executing model code.
"""
import argparse,hashlib,json,os,pathlib,shutil
p=argparse.ArgumentParser(description=__doc__)
p.add_argument('--matrix',type=pathlib.Path,required=True);p.add_argument('--destination',type=pathlib.Path,required=True);p.add_argument('--index',type=pathlib.Path,required=True)
a=p.parse_args();records=[]
for run in sorted(a.matrix.iterdir()):
 if not run.is_dir() or not (run/'receipt.json').exists():continue
 receipt=json.loads((run/'receipt.json').read_text())
 if 'elapsed_seconds' not in receipt:continue
 for directory,dirs,names in os.walk(run):
  dirs[:]=[d for d in dirs if not d.startswith('.') and d not in {'sessions','node_modules','go','pkg','cache','mod','sumdb','__pycache__'}]
  for name in names:
   src=pathlib.Path(directory)/name
   if src.suffix not in {'.js','.sql','.sh','.py'} or src.is_symlink():continue
   if src.stat().st_size>200000:continue
   rel=src.relative_to(run);dst=a.destination/run.name/rel
   dst.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(src,dst)
   records.append({'run':run.name,'path':str(dst),'source_path':str(src),'bytes':src.stat().st_size,'sha256':hashlib.sha256(src.read_bytes()).hexdigest()})
a.index.write_text(json.dumps({'scope':'model-authored source only; no claim every script succeeded; inspect the run receipts and independent grades','artifacts':records},indent=2)+'\n')
print(json.dumps({'files':len(records),'bytes':sum(r['bytes'] for r in records)}))
