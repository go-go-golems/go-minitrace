#!/usr/bin/env python3
"""Independent audit of native CommandExecution identities, argv and outcomes.
Usage: python3 08-audit-execution-coverage.py sources.txt archive-root
Never evaluates transcript commands; output contains counts/IDs, not private text.
"""
import hashlib
import json
import pathlib
import sys

archives = {}
for path in pathlib.Path(sys.argv[2]).rglob('*.minitrace.json'):
    value = json.loads(path.read_text())
    if value['id'] in archives:
        raise RuntimeError('duplicate archive identity')
    archives[value['id']] = value
results = []
for source in pathlib.Path(sys.argv[1]).read_text().splitlines():
    if not source.strip() or source.startswith('#'):
        continue
    data = pathlib.Path(source).read_bytes()
    records = [json.loads(line) for line in data.splitlines()]
    sid = next(r['payload']['id'] for r in records if r.get('type') == 'session_meta')
    expected = {}
    native_ids = set()
    for line, record in enumerate(records, 1):
        payload = record.get('payload', {})
        item = payload.get('item', {})
        if record.get('type') == 'event_msg' and payload.get('type') in ('item_started', 'item_completed') and item.get('type') == 'CommandExecution':
            native_ids.add(item['id'])
            if payload['type'] == 'item_completed':
                expected.setdefault(item['id'], []).append((line, item))
    actual = {}
    for call in archives[sid]['tool_calls']:
        metadata = call.get('framework_metadata') or {}
        if 'native_execution_id' in metadata:
            actual.setdefault(metadata['native_execution_id'], []).append(call)
    missing = sorted(set(expected)-set(actual))
    duplicates = sorted(k for k, calls in actual.items() if len(calls) != 1)
    invented = sorted(set(actual)-native_ids)
    mismatches = []
    failures = 0
    for eid, native_records in expected.items():
        if eid not in actual or len(actual[eid]) != 1:
            continue
        call = actual[eid][0]
        metadata = call['framework_metadata']
        output = call['output']
        item = native_records[-1][1]
        if metadata.get('argv') != item.get('command'):
            mismatches.append(dict(id=eid, field='argv'))
        codes = {r.get('exit_code') for _, r in native_records}
        if len(codes) == 1 and all(isinstance(c, int) for c in codes):
            code = next(iter(codes))
            if output.get('exit_code') != code or output.get('success') is not (code == 0):
                mismatches.append(dict(id=eid, field='outcome'))
            failures += code != 0
        text = item.get('aggregated_output', item.get('stdout', '') + item.get('stderr', ''))
        if output.get('truncated'):
            if output.get('full_hash') != 'sha256:' + hashlib.sha256(text.encode()).hexdigest() or output.get('full_bytes') != len(text.encode()):
                mismatches.append(dict(id=eid, field='full_output_hash_or_size'))
        elif output.get('result') != text:
            mismatches.append(dict(id=eid, field='output_text'))
        represented = {r['source_line'] for r in metadata.get('execution_sources', [])}
        if not {line for line, _ in native_records}.issubset(represented):
            mismatches.append(dict(id=eid, field='source_references'))
    row = dict(session_id=sid, source_sha256=hashlib.sha256(data).hexdigest(), native_completed_executions=len(expected),
               normalized_executions=len(actual), native_failed_executions=failures,
               missing=missing, invented=invented, duplicates=duplicates, mismatches=mismatches)
    row['passed'] = not (missing or invented or duplicates or mismatches)
    results.append(row)
print(json.dumps(results, indent=2))
sys.exit(0 if all(r['passed'] for r in results) else 1)
