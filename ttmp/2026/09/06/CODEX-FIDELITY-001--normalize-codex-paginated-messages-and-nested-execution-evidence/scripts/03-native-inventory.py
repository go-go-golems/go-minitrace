#!/usr/bin/env python3
"""Read native sources without evaluating transcript content; emit counts/anchors only.
Usage: python3 03-native-inventory.py /private/sources.txt
"""
import collections
import hashlib
import json
import pathlib
import sys

rows = []
for source in pathlib.Path(sys.argv[1]).read_text().splitlines():
    if not source.strip() or source.startswith('#'):
        continue
    data = pathlib.Path(source).read_bytes()
    counts = collections.Counter()
    ids = collections.defaultdict(set)
    anchors = {}
    meta = {}
    for line, raw in enumerate(data.splitlines(), 1):
        record = json.loads(raw)
        payload = record.get('payload', {})
        kind = record.get('type')
        if kind == 'session_meta':
            meta = {k: payload.get(k) for k in ('id', 'cli_version', 'history_mode', 'cwd')}
        if kind == 'response_item' and payload.get('type') == 'message':
            counts['response_messages'] += 1
        if kind == 'response_item' and payload.get('type') == 'custom_tool_call_output' and isinstance(payload.get('output'), list):
            counts['array_outputs'] += 1
            anchors.setdefault('array_output', {'line': line, 'call_id': payload.get('call_id')})
        if kind == 'event_msg' and payload.get('type') == 'item_completed':
            item = payload.get('item', {})
            item_type = item.get('type', 'unknown')
            counts[item_type] += 1
            if item.get('id'):
                ids[item_type].add(item['id'])
            anchor = {'line': line, 'id': item.get('id')}
            anchors.setdefault(item_type, anchor)
            if item_type == 'CommandExecution':
                code = item.get('exit_code')
                counts['executions_with_exit_code'] += code is not None
                if isinstance(code, int) and code != 0:
                    counts['nonzero_execution_events'] += 1
                    anchors.setdefault('failed_execution', dict(anchor, exit_code=code))
    rows.append(dict(source=source, sha256=hashlib.sha256(data).hexdigest(),
                     metadata=meta, counts=dict(counts),
                     unique_item_ids={k: len(v) for k, v in ids.items()}, anchors=anchors))
print(json.dumps(rows, indent=2))
