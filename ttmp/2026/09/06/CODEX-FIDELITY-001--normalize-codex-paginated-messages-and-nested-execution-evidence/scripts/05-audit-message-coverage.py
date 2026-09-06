#!/usr/bin/env python3
"""Audit native message coverage independently of converter reconciliation.

Usage: python3 05-audit-message-coverage.py sources.txt archive-root
All native message lines must be represented by a source reference; every
non-null tool linkage must name an assistant turn and be reciprocal.
"""
import json
import pathlib
import sys

archives = {}
for path in pathlib.Path(sys.argv[2]).rglob('*.minitrace.json'):
    archive = json.loads(path.read_text())
    if archive['id'] in archives:
        raise RuntimeError('duplicate archive ID: ' + archive['id'])
    archives[archive['id']] = archive
results = []
for source in pathlib.Path(sys.argv[1]).read_text().splitlines():
    if not source.strip() or source.startswith('#'):
        continue
    records = [json.loads(line) for line in pathlib.Path(source).read_text().splitlines()]
    sid = next(r['payload']['id'] for r in records if r.get('type') == 'session_meta')
    expected = set()
    for line, record in enumerate(records, 1):
        payload = record.get('payload', {})
        kind = record.get('type')
        if kind == 'response_item' and payload.get('type') == 'message' and payload.get('role') in ('user', 'assistant', 'system', 'developer'):
            expected.add(line)
        if kind == 'event_msg' and (
            payload.get('type') in ('user_message', 'agent_message') or
            (payload.get('type') == 'item_completed' and payload.get('item', {}).get('type') in ('UserMessage', 'AgentMessage'))
        ):
            expected.add(line)
    archive = archives[sid]
    actual = []
    for turn in archive['turns']:
        metadata = turn.get('framework_metadata') or {}
        actual.extend(s['source_line'] for s in metadata.get('message_sources', []))
    turns = {t['index']: t for t in archive['turns']}
    invalid = []
    for call in archive['tool_calls']:
        index = call.get('emitting_turn_index')
        if index is not None and (index not in turns or turns[index]['role'] != 'assistant'
                                  or call['id'] not in turns[index]['tool_calls_in_turn']):
            invalid.append(call['id'])
    row = dict(session_id=sid, normalized_turns=len(turns), native_message_records=len(expected),
               represented_message_records=len(actual), missing_source_lines=sorted(expected-set(actual)),
               invented_source_lines=sorted(set(actual)-expected), duplicate_source_lines=len(actual)-len(set(actual)),
               invalid_links=invalid)
    row['passed'] = not (row['missing_source_lines'] or row['invented_source_lines'] or row['duplicate_source_lines'] or invalid)
    results.append(row)
print(json.dumps(results, indent=2))
sys.exit(0 if all(r['passed'] for r in results) else 1)
