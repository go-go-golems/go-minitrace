#!/usr/bin/env python3
"""Preserve generated runnable sources and hashes without copying raw transcripts."""
import argparse,hashlib,json,shutil
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--runs',type=Path,required=True);p.add_argument('--destination',type=Path,required=True);a=p.parse_args();rows=[]
for run in sorted(a.runs.iterdir()):
 if not run.is_dir():continue
 for src in sorted(run.rglob('*')):
  if not src.is_file() or src.is_symlink() or src.suffix not in {'.js','.sql','.py','.sh'} or 'sessions' in src.relative_to(run).parts:continue
  rel=src.relative_to(a.runs);dest=a.destination/rel;body=src.read_bytes();digest=hashlib.sha256(body).hexdigest()
  if dest.exists() and dest.read_bytes()!=body:raise ValueError('Refusing changed artifact overwrite: '+str(dest))
  dest.parent.mkdir(parents=True,exist_ok=True);dest.write_bytes(body);rows.append({'path':str(rel),'bytes':len(body),'sha256':digest})
a.destination.mkdir(parents=True,exist_ok=True);(a.destination/'manifest.json').write_text(json.dumps(rows,indent=2)+'\n');print(json.dumps({'files':len(rows),'bytes':sum(r['bytes'] for r in rows)}))
