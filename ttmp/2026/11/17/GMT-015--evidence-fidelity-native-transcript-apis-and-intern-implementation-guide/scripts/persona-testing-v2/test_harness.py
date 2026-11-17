#!/usr/bin/env python3
"""No models: real OS negative probes, wrapper statuses and explicit-order grading."""
import json,sys,tempfile,unittest
from pathlib import Path
from common import probe,profile
from record_command import execute
from evaluate import grade
from run_cohort import prepare_run_directories

class HarnessTests(unittest.TestCase):
 def test_all_run_directories_exist_before_workers(self):
  with tempfile.TemporaryDirectory(prefix='gmt015-v2-runs-') as tmp:
   root=Path(tmp);specs=[(g,'model','high',1) for g in 'ABCD']
   prepare_run_directories(root,'cohort',specs)
   paths=list((root/'runs/cohort').iterdir())
   self.assertEqual(len(paths),4)
   for own in paths:
    siblings=[p for p in (root/'runs').glob('*/*') if p.is_dir() and p!=own]
    self.assertEqual(len(siblings),3)
 def test_nonobject_answers_are_validation_errors(self):
  for value in [None,[],42,'answer']:
   with self.subTest(value=value),self.assertRaisesRegex(ValueError,'JSON object'):
    grade(value,'A',[],{'expected':{}},Path('/tmp'),{})
 def test_d_replay_checks_session_identity(self):
  with tempfile.TemporaryDirectory(prefix='gmt015-v2-grade-') as tmp:
   root=Path(tmp);stdout=root/'query.json';stdout.write_text(json.dumps({'session_id':'wrong','tool_count':0}))
   stderr=root/'stderr';stderr.write_text('bad.js')
   helpout=root/'help';helpout.write_text('--source --archive-glob')
   commands=[{'exit_code':0,'stdout_path':str(stdout),'stderr_path':str(stderr)} for _ in range(3)]
   commands.append({'exit_code':0,'stdout_path':str(helpout),'stderr_path':str(stderr)})
   rows=[{'session_id':str(i),'tool_count':0} for i in range(3)]
   receipt={'host_replay':{'commands':commands,'selected_session_ids':[r['session_id'] for r in rows]}}
   answer={'schema_version':2,'task':'D','good_command_succeeded':True,'negative_default_supported':False}
   checks=grade(answer,'D',rows,{'expected':{}},root,receipt)
   self.assertTrue(all(c['passed'] for c in checks if c['key'].endswith('.tool_count')))
   identities=[c for c in checks if c['key'].endswith('.session_id')]
   self.assertEqual(len(identities),3);self.assertTrue(all(not c['passed'] for c in identities))
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
    r=execute([sys.executable,'-c',f'raise SystemExit({code})'],p,p/f'{code}.json',10);self.assertEqual(r['exit_code'],code);self.assertTrue(Path(r['stdout_path']).is_absolute())
 def test_order_contract_and_dataset_separation(self):
  fixture={'expected':{'unicode_bytes':4,'clipped_text':'ab','clipped_complete':False,'strict_rows':0,'strict_error':True,'manifest_duplicate_count':1,'manifest_conflict_count':1}}
  rows=[{'session_id':f,'framework':f} for f in ['claude-code','codex','pi']]
  answer={'schema_version':2,'task':'B','synthetic':dict(fixture['expected'],archive_source_ordinals=[0,1,2],chronological_source_ordinals=[1,2,0]),'real_sessions':[{'session_id':r['session_id']} for r in rows]}
  self.assertTrue(all(c['passed'] for c in grade(answer,'B',rows,fixture,Path('/tmp'),{})))
  answer['synthetic']['chronological_source_ordinals']=[0,1,2]
  self.assertFalse(next(c['passed'] for c in grade(answer,'B',rows,fixture,Path('/tmp'),{}) if c['key']=='synthetic.chronological_source_ordinals'))
if __name__=='__main__':unittest.main()
