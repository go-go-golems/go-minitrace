#!/usr/bin/env python3
"""Check the paginated-fidelity fixture's converted archive without executing input.

Usage: python3 04-check-synthetic-baseline.py /path/to/fidelity-synthetic.minitrace.json
Exits nonzero while required fidelity assertions remain unmet. This is an
independent before/after oracle, not a replacement for package regression tests.
"""
import json
import sys

session = json.load(open(sys.argv[1]))
turns = session.get('turns', [])
calls = session.get('tool_calls', [])
checks = []


def check(name, passed, detail):
    checks.append(dict(name=name, passed=bool(passed), detail=detail))


check('native_identity', session['id'] == 'fidelity-synthetic', session['id'])
check('five_deduplicated_messages', len(turns) == 5, len(turns))
check('repeated_continue_retained', sum(t['content'] == 'continue' for t in turns) == 2,
      [t['content'] for t in turns if t['role'] == 'user'])
valid_indexes = {t['index'] for t in turns if t['role'] == 'assistant'}
orphans = [c['id'] for c in calls if c.get('emitting_turn_index') is not None
           and c['emitting_turn_index'] not in valid_indexes]
check('no_orphan_associations', not orphans, orphans)
malformed = [c['id'] for c in calls if 'map[' in (c.get('output', {}).get('result') or '')]
check('typed_outputs_decoded', not malformed, malformed)
failed = [c for c in calls if c.get('output', {}).get('exit_code') == 7
          and c.get('input', {}).get('command') ==
          'printf first > first.txt; printf second >> second.txt; exit 7']
check('one_authoritative_failed_execution', len(failed) == 1, len(failed))
ok = [c for c in calls if c.get('input', {}).get('command') == 'printf ok']
check('identical_commands_distinct_ids', len(ok) == 2 and len({c['id'] for c in ok}) == 2,
      [c['id'] for c in ok])
missing = next((c for c in calls if c['id'] == 'missing-output'), None)
check('missing_outcome_not_success', missing is not None and missing['output']['success'] is None,
      missing['output'] if missing else None)
early = next((c for c in calls if c['id'] == 'early-output'), None)
check('output_before_call_preserved', early is not None and early['output']['exit_code'] == 0
      and early['output']['result'] == 'direct output', early['output'] if early else None)
print(json.dumps(checks, indent=2))
sys.exit(0 if all(c['passed'] for c in checks) else 1)
