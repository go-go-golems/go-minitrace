#!/usr/bin/env python3
"""Preserve the specifically identified out-of-scope model redirection outputs.
Only known, untracked files are moved; never remove tracked product files.
"""
import argparse,hashlib,json,pathlib,shutil,subprocess
p=argparse.ArgumentParser(description=__doc__);p.add_argument('--repo',type=pathlib.Path,required=True);p.add_argument('--destination',type=pathlib.Path,default=pathlib.Path('/private/tmp/gmt015-persona/scope-artifacts'));p.add_argument('--receipt',type=pathlib.Path,required=True);a=p.parse_args()
tracked=set(subprocess.check_output(['git','ls-files'],cwd=a.repo,text=True).splitlines());a.destination.mkdir(parents=True,exist_ok=True);rows=[]
for name in ['err.log',*[f'err{i}.log' for i in range(2,10)],'out_scan.json','out_scan.txt','out-inspect-cp.json']:
 src=a.repo/name
 if not src.exists():continue
 if name in tracked:raise RuntimeError('refusing tracked file '+name)
 dst=a.destination/name
 if dst.exists():raise RuntimeError('refusing overwrite '+str(dst))
 raw=src.read_bytes();shutil.move(str(src),str(dst))
 rows.append({'original_path':str(src),'preserved_path':str(dst),'bytes':len(raw),'sha256':hashlib.sha256(raw).hexdigest(),'run':'C-deepseek-4.1-flash-low-r2' if name=='out-inspect-cp.json' else 'C-glm-5.3-flash-high-r2'})
a.receipt.write_text(json.dumps({'scope':'two agents used relative shell redirections after cd into the read-only-by-prompt repository; actual untracked writes violated workspace instructions; product tracked file hashes remain unchanged','files':rows,'moved_not_deleted':True,'next_harness':'run a hash-pinned binary from the run directory and enforce read-only repo mounts'},indent=2)+'\n');print(json.dumps({'preserved_files':len(rows),'runs':sorted({r['run']for r in rows})}))
