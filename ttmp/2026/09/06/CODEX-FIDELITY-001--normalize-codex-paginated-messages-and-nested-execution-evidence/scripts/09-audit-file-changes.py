#!/usr/bin/env python3
"""Read-only native FileChange audit; reports counts/IDs, never private bodies."""
import json
import pathlib
import sys

archives = {s['id']: s for p in pathlib.Path(sys.argv[2]).rglob('*.minitrace.json') for s in [json.loads(p.read_text())]}
results = []
for source in pathlib.Path(sys.argv[1]).read_text().splitlines():
    if not source or source.startswith('#'):
        continue
    records = [json.loads(line) for line in pathlib.Path(source).read_text().splitlines()]
    sid = next(r['payload']['id'] for r in records if r.get('type') == 'session_meta')
    expected = {}
    for record in records:
        payload = record.get('payload', {})
        item = payload.get('item', {})
        if record.get('type') == 'event_msg' and payload.get('type') == 'item_completed' and item.get('type') == 'FileChange':
            targets = []
            for path, change in item['changes'].items():
                operation = {'add': 'NEW', 'update': 'MODIFY', 'delete': 'DELETE'}[change['type']]
                if change.get('move_path'):
                    targets.extend([(path, 'DELETE'), (change['move_path'], 'NEW')])
                else:
                    targets.append((path, operation))
            expected[item['id']] = (sorted(targets), item['status'])
    actual = {}
    problems = []
    for call in archives[sid]['tool_calls']:
        metadata = call.get('framework_metadata') or {}
        if 'native_file_change_id' not in metadata:
            continue
        identity = metadata['native_file_change_id']
        if identity in actual:
            problems.append({'id': identity, 'field': 'duplicate_identity'})
        actual[identity] = call
    for identity, (targets, status) in expected.items():
        if identity not in actual:
            problems.append({'id': identity, 'field': 'missing'})
            continue
        call = actual[identity]
        observed = call['input']['file_targets']
        if sorted((t['path'], t['operation_type']) for t in observed) != targets:
            problems.append({'id': identity, 'field': 'targets'})
        if status == 'completed' and any(t['success'] is not True or t['status'] != 'confirmed' for t in observed):
            problems.append({'id': identity, 'field': 'effect_outcome'})
        if any(t['evidence_kind'] != 'native_file_change' or not t.get('source_reference') for t in observed):
            problems.append({'id': identity, 'field': 'provenance'})
    for identity in set(actual)-set(expected):
        problems.append({'id': identity, 'field': 'invented_identity'})
    results.append(dict(session_id=sid, native_file_changes=len(expected), native_file_targets=sum(len(v[0]) for v in expected.values()), normalized_file_changes=len(actual), problems=problems, passed=not problems))
print(json.dumps(results, indent=2))
sys.exit(0 if all(r['passed'] for r in results) else 1)
