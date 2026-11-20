#!/usr/bin/env python3
"""Capture only public model capability metadata; never dump auth/config secrets."""
import argparse,json,pathlib,subprocess,hashlib
p=argparse.ArgumentParser();p.add_argument('--output',type=pathlib.Path,required=True);a=p.parse_args()
s=json.loads((pathlib.Path.home()/'.pi/agent/models-store.json').read_text())
models=[{k:m.get(k) for k in ['id','api','reasoning','thinkingLevelMap','compat']} for m in s.get('lunaroute',{}).get('models',[]) if m.get('id') in ['glm-5.3-flash','deepseek-4.1-flash']]
a.output.write_text(json.dumps({'pi_version':subprocess.check_output(['pi','--version'],text=True).strip(),'models':models,'extension_sha256':{str(q.relative_to(pathlib.Path.home())):hashlib.sha256(q.read_bytes()).hexdigest() for q in [(pathlib.Path.home()/'.pi/agent/npm/node_modules/@lunaroute/pi-extension'/r) for r in ['package.json','src/index.ts','src/lunaroute.ts']]},'scope':'local advertised compatibility, not wire capture or proof of backend behavior','billing':'extension assigns zero cost rates; token usage cost is not independent proof of free billing'},indent=2)+'\n')
