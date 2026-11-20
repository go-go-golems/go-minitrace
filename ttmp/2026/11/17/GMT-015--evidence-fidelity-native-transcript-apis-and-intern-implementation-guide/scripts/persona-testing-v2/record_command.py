#!/usr/bin/env python3
"""Execute explicit argv with timeout; preserve zero exit status and exact receipts."""
import argparse,hashlib,json,os,signal,subprocess,time
from pathlib import Path

def execute(argv,cwd,receipt,timeout=90):
 receipt=Path(receipt).resolve();receipt.parent.mkdir(parents=True,exist_ok=True)
 if receipt.exists():raise ValueError('Refusing to overwrite command receipt '+str(receipt))
 start=time.time();timed_out=False
 with receipt.with_suffix('.stdout').open('wb') as out,receipt.with_suffix('.stderr').open('wb') as err:
  child=subprocess.Popen(argv,cwd=cwd,stdout=out,stderr=err,start_new_session=True)
  try:code=child.wait(timeout=timeout)
  except subprocess.TimeoutExpired:
   timed_out=True;os.killpg(child.pid,signal.SIGTERM)
   try:code=child.wait(timeout=3)
   except subprocess.TimeoutExpired:os.killpg(child.pid,signal.SIGKILL);code=child.wait()
 data={'argv':argv,'cwd':str(Path(cwd).resolve()),'started_at':start,'elapsed_seconds':round(time.time()-start,3),'exit_code':code,'timed_out':timed_out,'stdout_path':str(receipt.with_suffix('.stdout')),'stderr_path':str(receipt.with_suffix('.stderr'))}
 for stream in ('stdout','stderr'):data[stream+'_sha256']=hashlib.sha256(receipt.with_suffix('.'+stream).read_bytes()).hexdigest()
 receipt.write_text(json.dumps(data,indent=2)+'\n');return data

def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--receipt',type=Path,required=True);p.add_argument('--cwd',default='.');p.add_argument('--timeout',type=int,default=90);p.add_argument('command',nargs=argparse.REMAINDER);a=p.parse_args();cmd=a.command[1:] if a.command[:1]==['--'] else a.command
 if not cmd:p.error('explicit command argv required after --')
 d=execute(cmd,a.cwd,a.receipt,a.timeout);print(json.dumps(d));raise SystemExit(d['exit_code'] if d['exit_code']>=0 else 128-d['exit_code'])
if __name__=='__main__':main()
