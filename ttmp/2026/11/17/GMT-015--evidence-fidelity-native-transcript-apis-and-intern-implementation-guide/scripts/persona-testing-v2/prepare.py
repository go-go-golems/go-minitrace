#!/usr/bin/env python3
"""Build once and freeze V2 bundle. Never invokes Pi/models. Destination must be fresh."""
import argparse,json,shutil,subprocess,sys
from pathlib import Path
from common import inventory,load,platform_guard,save,sha
from contracts import SCHEMAS

def prepare(repo,corpus,fixtures,bundle):
 platform_guard();repo=Path(repo).resolve();corpus=Path(corpus).resolve();fixtures=Path(fixtures).resolve();bundle=Path(bundle).resolve()
 if bundle.exists():raise ValueError('Fresh bundle directory required; refusing overwrite')
 bundle.mkdir(parents=True)
 for name in ['bin','control','private-oracle','runs','public']:(bundle/name).mkdir()
 subprocess.run(['go','build','-o',str(bundle/'bin/go-minitrace'),'./cmd/go-minitrace'],cwd=repo,check=True)
 shutil.copy2(Path(__file__).with_name('record_command.py'),bundle/'bin/record_command.py')
 for name in ['raw','archives']:shutil.copytree(corpus/name,bundle/'corpus'/name)
 # Explicit fixture allowlist excludes the oracle and previous agent output.
 (bundle/'fixtures').mkdir()
 for name in ['archive.json','archive-copy.json','archive-conflict.json','native.jsonl']:shutil.copy2(fixtures/name,bundle/'fixtures'/name)
 original=load(corpus/'manifest.json');rows=[]
 for item in original['rows']:
  r=dict(item)
  for key in ['native_path','archive_path']:r[key]=str(bundle/'corpus'/Path(r[key]).relative_to(corpus))
  if sha(r['native_path'])!=r['native_sha256'] or sha(r['archive_path'])!=r['archive_sha256']:raise ValueError('Copied corpus hash mismatch')
  rows.append(r)
 save(bundle/'private-oracle/corpus.json',{'rows':rows});shutil.copy2(fixtures/'oracle.json',bundle/'private-oracle/fixtures.json');save(bundle/'public/answer-schema.json',SCHEMAS)
 gitdir=subprocess.check_output(['git','rev-parse','--git-common-dir'],cwd=repo,text=True).strip();gitdir=(repo/gitdir).resolve()
 protected=[repo,gitdir,corpus,fixtures,bundle/'bin',bundle/'control',bundle/'private-oracle',bundle/'corpus',bundle/'fixtures',bundle/'public']
 denied=[bundle/'private-oracle',repo/'ttmp',corpus/'manifest.json',fixtures/'oracle.json']
 pinned=inventory([bundle/'bin',bundle/'corpus',bundle/'fixtures',bundle/'public'])
 config={'schema_version':2,'repo':str(repo),'bundle':str(bundle),'revision':subprocess.check_output(['git','rev-parse','HEAD'],cwd=repo,text=True).strip(),'binary':str(bundle/'bin/go-minitrace'),'binary_sha256':sha(bundle/'bin/go-minitrace'),'source_diff_sha256':__import__('hashlib').sha256(subprocess.check_output(['git','diff','HEAD'],cwd=repo)).hexdigest(),'protected_paths':[str(p) for p in protected],'denied_read_paths':[str(p) for p in denied],'hashes':pinned,'limits':'Write protection for listed canonical paths and oracle read denial only; not full OS containment. Network/provider state outside protected paths remains available.'}
 save(bundle/'control/inputs.json',config);return config

def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--repo',type=Path,required=True);p.add_argument('--corpus',type=Path,default=Path('/private/tmp/gmt015-persona/corpus'));p.add_argument('--fixtures',type=Path,default=Path('/private/tmp/gmt015-persona/fixtures'));p.add_argument('--bundle',type=Path,required=True);a=p.parse_args();c=prepare(a.repo,a.corpus,a.fixtures,a.bundle);print(json.dumps({'bundle':c['bundle'],'binary_sha256':c['binary_sha256']}))
if __name__=='__main__':main()
