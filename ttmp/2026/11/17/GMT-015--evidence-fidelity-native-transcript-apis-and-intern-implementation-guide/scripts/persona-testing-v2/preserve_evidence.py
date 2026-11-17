#!/usr/bin/env python3
"""Archive private experiment bundles outside Git; publish only the integrity receipt."""
import argparse,hashlib,json,tarfile
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--bundle',type=Path,action='append',required=True);p.add_argument('--archive',type=Path,required=True);p.add_argument('--receipt',type=Path,required=True);a=p.parse_args()
if a.archive.exists():raise ValueError('Refusing archive overwrite')
a.archive.parent.mkdir(parents=True,exist_ok=True)
with tarfile.open(a.archive,'w:gz') as tf:
 for bundle in a.bundle:tf.add(bundle,arcname=bundle.name)
h=hashlib.sha256()
with a.archive.open('rb') as f:
 for block in iter(lambda:f.read(1024*1024),b''):h.update(block)
r={'archive':str(a.archive.resolve()),'bytes':a.archive.stat().st_size,'sha256':h.hexdigest(),'bundles':[str(b.resolve()) for b in a.bundle],'private':True};a.receipt.parent.mkdir(parents=True,exist_ok=True);a.receipt.write_text(json.dumps(r,indent=2)+'\n');print(json.dumps(r))
