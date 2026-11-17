"""Shared V2 isolation and immutable-input helpers. No model calls here."""
import hashlib,json,os,platform,subprocess,sys
from pathlib import Path

def sha(path):return hashlib.sha256(Path(path).read_bytes()).hexdigest()
def load(path):return json.loads(Path(path).read_text())
def save(path,value):
 path=Path(path);path.parent.mkdir(parents=True,exist_ok=True);temp=path.with_suffix(path.suffix+'.tmp');temp.write_text(json.dumps(value,indent=2)+'\n');temp.replace(path)
def platform_guard():
 if platform.system()!='Darwin' or not Path('/usr/bin/sandbox-exec').exists():raise RuntimeError('OS-enforced V2 isolation unavailable: this implementation requires macOS sandbox-exec; no prompt-only fallback')
def profile(protected,denied_reads):
 platform_guard();lines=['(version 1)','(allow default)']
 for path in protected:lines.append('(deny file-write* (subpath '+json.dumps(str(Path(path).resolve()))+'))')
 for path in denied_reads:lines.append('(deny file-read* (subpath '+json.dumps(str(Path(path).resolve()))+'))')
 return '\n'.join(lines)+'\n'
def inventory(paths):
 result={}
 for root in paths:
  root=Path(root)
  for path in ([root] if root.is_file() else sorted(root.rglob('*'))):
   if path.is_file() and not path.is_symlink():result[str(path.resolve())]=sha(path)
 return result

def verify(bundle):
 expected=load(Path(bundle)/'control/inputs.json')
 for path,digest in expected['hashes'].items():
  if sha(path)!=digest:raise RuntimeError('Pinned input drift: '+path)
 return expected

def probe(profile_path,protected,oracle,workspace):
 """Fail closed unless direct and nested child writes fail on every protected root."""
 platform_guard();workspace=Path(workspace);workspace.mkdir(parents=True,exist_ok=True);checks=[]
 for i,root in enumerate(protected):
  root=Path(root).resolve();target=(root if root.is_dir() else root.parent)/('.gmt015-denied-probe-'+str(os.getpid()))
  command=[sys.executable,'-c','import pathlib,subprocess,sys; p=sys.argv[1]; direct=False\ntry: pathlib.Path(p).write_text("probe"); direct=True\nexcept OSError: pass\nr=subprocess.run(["/bin/sh","-c",\'printf probe > "$1"\',"probe",p],capture_output=True); sys.exit(1 if direct or r.returncode==0 else 0)',str(target)]
  result=subprocess.run(['/usr/bin/sandbox-exec','-f',str(profile_path)]+command,capture_output=True,text=True)
  allowed=target.exists()
  if allowed:target.unlink() # Only our newly created probe, never existing data.
  checks.append({'protected_path':str(root),'direct_and_nested_denied':result.returncode==0 and not allowed,'probe_exit':result.returncode,'stderr':result.stderr[-1000:]})
  if result.returncode!=0 or allowed:raise RuntimeError('Negative write probe failed: '+str(root)+' '+result.stderr)
 # Resolve an alias from writable scratch: deny rules must follow the target.
 alias=workspace/'protected-alias'
 if not alias.exists():alias.symlink_to(Path(protected[0]).resolve(),target_is_directory=True)
 target=alias/('.gmt015-alias-probe-'+str(os.getpid()))
 r=subprocess.run(['/usr/bin/sandbox-exec','-f',str(profile_path),sys.executable,'-c','import pathlib,sys; pathlib.Path(sys.argv[1]).write_text("probe")',str(target)],capture_output=True,text=True)
 if target.exists():target.unlink();raise RuntimeError('Symlink write bypass allowed')
 if r.returncode==0:raise RuntimeError('Symlink denial probe did not fail')
 checks.append({'symlink_target':str(alias.resolve()),'write_denied':True})
 r=subprocess.run(['/usr/bin/sandbox-exec','-f',str(profile_path),sys.executable,'-c','import pathlib,sys; pathlib.Path(sys.argv[1]).read_bytes()',str(oracle)],capture_output=True,text=True)
 if r.returncode==0:raise RuntimeError('Oracle read was permitted')
 checks.append({'oracle_read_denied':True})
 r=subprocess.run(['/usr/bin/sandbox-exec','-f',str(profile_path),sys.executable,'-c','import pathlib,sys; pathlib.Path(sys.argv[1]).write_text("ok")',str(workspace/'allowed-probe')],capture_output=True,text=True)
 if r.returncode!=0:raise RuntimeError('Writable workspace probe failed: '+r.stderr)
 checks.append({'workspace_write_allowed':True});return checks
