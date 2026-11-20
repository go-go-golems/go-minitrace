#!/usr/bin/env python3
"""Record independent validation exit codes; do not collapse them into last-command status."""
import argparse,json,os,subprocess,time
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--output',type=Path,required=True);a=p.parse_args();a.output.mkdir(parents=True,exist_ok=True)
commands={"build":["go","build","./..."],"vet":["go","vet","./..."],"help":["go","run","./cmd/go-minitrace","help","--all"],"glazed-lint":["go","vet","-vettool=/tmp/glazed-lint","-glazedclilint.allow-paths=pkg/analysis/,pkg/cli/,pkg/cmds/fields/,pkg/cmds/logging/,pkg/cmds/sources/,pkg/help/","./cmd/...","./pkg/..."]}
rows=[]
for name,argv in commands.items():
 start=time.time()
 with (a.output/(name+'.log')).open('wb') as stream:r=subprocess.run(argv,stdout=stream,stderr=subprocess.STDOUT,env=dict(os.environ,TMPDIR='/private/tmp',GOWORK='off'))
 rows.append(dict(name=name,argv=argv,exit_code=r.returncode,elapsed_seconds=round(time.time()-start,3)))
(a.output/'validation.json').write_text(json.dumps(rows,indent=2)+'\n');print(json.dumps(rows,indent=2))
raise SystemExit(any(r['exit_code'] for r in rows))
