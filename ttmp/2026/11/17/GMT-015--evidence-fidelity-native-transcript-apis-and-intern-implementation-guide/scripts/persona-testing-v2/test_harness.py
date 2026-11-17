#!/usr/bin/env python3
"""No models: real OS negative probes, wrapper statuses and explicit-order grading."""
import json,sys,tempfile,unittest
from pathlib import Path
from common import probe,profile
from record_command import execute
from evaluate import grade

class HarnessTests(unittest.TestCase):
 def test_enforced_paths_and_nested_children(self):
  with tempfile.TemporaryDirectory(prefix='gmt015-v2-isolation-') as tmp:
   root=Path(tmp).resolve();protected=[]
   for name in ['repo','corpus','fixtures','bin','private']:
    p=root/name;p.mkdir();protected.append(p)
   oracle=root/'private/oracle.json';oracle.write_text('{"secret":"test-only"}')
   policy=root/'policy.sb';policy.write_text(profile(protected,[root/'private']))
   checks=probe(policy,protected,oracle,root/'work');self.assertEqual(len(checks),8)
 def test_exit_zero_and_nonzero(self):
  with tempfile.TemporaryDirectory(prefix='gmt015-v2-status-') as tmp:
   p=Path(tmp)
   for code in [0,7]:
    r=execute([sys.executable,'-c',f'raise SystemExit({code})'],p,p/f'{code}.json',10);self.assertEqual(r['exit_code'],code)
 def test_order_contract_and_dataset_separation(self):
  fixture={'expected':{'unicode_bytes':4,'clipped_text':'ab','clipped_complete':False,'strict_rows':0,'strict_error':True,'manifest_duplicate_count':1,'manifest_conflict_count':1}}
  rows=[{'session_id':f,'framework':f} for f in ['claude-code','codex','pi']]
  answer={'schema_version':2,'task':'B','synthetic':dict(fixture['expected'],archive_source_ordinals=[0,1,2],chronological_source_ordinals=[1,2,0]),'real_sessions':[{'session_id':r['session_id']} for r in rows]}
  self.assertTrue(all(c['passed'] for c in grade(answer,'B',rows,fixture,Path('/tmp'),{})))
  answer['synthetic']['chronological_source_ordinals']=[0,1,2]
  self.assertFalse(next(c['passed'] for c in grade(answer,'B',rows,fixture,Path('/tmp'),{}) if c['key']=='synthetic.chronological_source_ordinals'))
if __name__=='__main__':unittest.main()
