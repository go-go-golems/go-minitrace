#!/usr/bin/env python3
"""Regression checks for evaluator type strictness and source-path canonicalization."""
import importlib.util,json,pathlib,tempfile,unittest
HERE=pathlib.Path(__file__).parent

def module(name):
 spec=importlib.util.spec_from_file_location(name,HERE/(name+'.py'));m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
G=module('grade');S=module('summarize_results')

class GraderTests(unittest.TestCase):
 def test_boolean_is_not_integer(self):
  self.assertFalse(G.same(True,1));self.assertFalse(G.same([False],[0]));self.assertTrue(G.same([0,1],[0,1]))
 def test_order_matters(self):self.assertFalse(G.same([1,2,0],[0,1,2]))
 def test_relative_paths_and_duplicate_aliases(self):
  import hashlib
  with tempfile.TemporaryDirectory() as tmp:
   path=pathlib.Path(tmp)/'raw'/'pi'/'one.jsonl';path.parent.mkdir(parents=True);raw=b'{"type":"message"}\n';path.write_bytes(raw)
   row={'native_path':str(path),'native_sha256':hashlib.sha256(raw).hexdigest(),'native_bytes':len(raw)}
   scan={'path':'pi/one.jsonl','sha256':row['native_sha256'],'bytes':len(raw),'lines':1,'hashScope':'whole_source'}
   answer={'exact_replay_possible':False,'real_scans':[scan]}
   self.assertTrue(all(c['passed'] for c in S.grade_c(answer,[row])))
   answer['real_scans'].append(dict(scan,path=str(path)))
   self.assertFalse(all(c['passed'] for c in S.grade_c(answer,[row])))
   answer['real_scans']=[dict(scan,path='one.jsonl')]
   self.assertFalse(all(c['passed'] for c in S.grade_c(answer,[row])))

if __name__=='__main__':unittest.main()
