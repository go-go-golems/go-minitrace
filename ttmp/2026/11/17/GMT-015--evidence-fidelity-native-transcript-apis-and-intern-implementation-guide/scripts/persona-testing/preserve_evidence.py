#!/usr/bin/env python3
"""Preserve private experiment evidence outside Git and temporary storage."""
import argparse,hashlib,json,pathlib,tarfile
p=argparse.ArgumentParser(description=__doc__);p.add_argument('--source',type=pathlib.Path,default=pathlib.Path('/private/tmp/gmt015-persona'));p.add_argument('--archive',type=pathlib.Path,required=True);p.add_argument('--receipt',type=pathlib.Path,required=True);a=p.parse_args()
a.archive.parent.mkdir(parents=True,exist_ok=True)
if a.archive.exists():raise SystemExit('Refusing to overwrite existing evidence archive')
skipped=[]
def select(info):
 parts=pathlib.PurePosixPath(info.name).parts
 if any(x.startswith('.env') or x in {'auth.json','credentials.json','__pycache__','.cache','node_modules'} for x in parts):
  skipped.append(info.name);return None
 return info
with tarfile.open(a.archive,'w:gz') as out:out.add(a.source,arcname='gmt015-persona',filter=select)
h=hashlib.sha256()
with a.archive.open('rb')as f:
 for block in iter(lambda:f.read(1<<20),b''):h.update(block)
r={'archive':str(a.archive),'sha256':h.hexdigest(),'bytes':a.archive.stat().st_size,'source':str(a.source),'scope':'private corpus snapshots, full Pi sessions/events, authored outputs, review receipts and normalized re-analysis; not uploaded or committed','skipped_cache_or_credential_paths':skipped}
a.receipt.write_text(json.dumps(r,indent=2)+'\n');print(json.dumps({k:r[k]for k in ['archive','sha256','bytes']}))
