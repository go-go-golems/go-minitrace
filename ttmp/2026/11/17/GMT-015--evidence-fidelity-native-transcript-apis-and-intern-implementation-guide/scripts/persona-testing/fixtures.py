#!/usr/bin/env python3
"""Generate synthetic persona inputs and a private, independently computed oracle.

Never give oracle.json to the tested agent. Generation uses no go-minitrace code.
"""
import argparse
import copy
import hashlib
import json
from pathlib import Path


def encoded(value):
    return (json.dumps(value, ensure_ascii=False, separators=(',', ':')) + '\n').encode()


def generate(out):
    out.mkdir(parents=True, exist_ok=True)
    archive = {
        'id': 'persona-synthetic', 'schema_version': '1.0', 'profile': 'synthetic',
        'classification': 'synthetic', 'title': 'Evidence fidelity fixture',
        'provenance': {'source_format': 'synthetic', 'original_session_id': 'native-persona'},
        'turns': [
            {'index': 0, 'role': 'assistant', 'content': 'Missing usage', 'usage': None},
            {'index': 1, 'role': 'assistant', 'content': 'Recorded zero usage',
             'usage': {'input_tokens': 0, 'output_tokens': 0}},
        ],
        'tool_calls': [],
    }
    for ident, second, status, success in [('z-first', 3, 'failed', False),
                                          ('a-second', 1, 'succeeded', True),
                                          ('m-third', 2, 'unknown', None)]:
        archive['tool_calls'].append({
            'id': ident, 'record_kind': 'execution', 'emitting_turn_index': 0,
            'timestamp': f'2026-01-01T00:00:0{second}Z', 'tool_name': 'synthetic_read',
            'operation_type': 'READ', 'input': {'arguments': {'label': ident}},
            'output': {'result': 'abé', 'status': status, 'success': success},
        })
    variant = copy.deepcopy(archive)
    variant['title'] = 'Conflicting content under the same ID'
    files = {'archive.json': encoded(archive), 'archive-copy.json': encoded(archive),
             'archive-conflict.json': encoded(variant)}
    records = [
        {'type': 'api-request-shape', 'shapeHash': 's', 'shape': {'system': [{'text': 'synthetic rules é'}]}},
        {'type': 'api-request-blob', 'hash': 'a', 'message': {'role': 'user', 'content': 'abé'}},
        {'type': 'api-request', 'id': 'r1', 'shapeHash': 's', 'keep': 0, 'tail': ['a']},
        {'type': 'api-request-blob', 'hash': 'b', 'message': {'role': 'assistant', 'content': 'second'}},
        {'type': 'api-request', 'id': 'r2', 'shapeHash': 's', 'base': 'r1', 'keep': 1, 'tail': ['b']},
        {'type': 'api-request', 'id': 'r3', 'shapeHash': 's', 'keep': 0, 'tail': ['missing']},
        {'type': 'assistant', 'uuid': 'old'},
        {'type': 'system', 'subtype': 'compact_boundary', 'uuid': 'boundary',
         'logicalParentUuid': 'attachment',
         'compactMetadata': {'preservedSegment': {'headUuid': 'old', 'tailUuid': 'attachment'}}},
        {'type': 'user', 'uuid': 'summary', 'parentUuid': 'boundary', 'isCompactSummary': True},
        {'type': 'attachment', 'uuid': 'attachment', 'parentUuid': 'summary'},
        {'type': 'assistant', 'uuid': 'new', 'parentUuid': 'attachment'},
    ]
    # Deliberately mix CRLF, LF, malformed JSON and a final line without LF.
    native = encoded(records[0]).replace(b'\n', b'\r\n')
    native += b''.join(encoded(row) for row in records[1:])
    native += b'{broken json}\n' + b'{"type":"last","exact":9007199254740993}'
    files['native.jsonl'] = native
    for name, content in files.items():
        (out / name).write_bytes(content)
    expected = {
        'unicode_bytes': len('abé'.encode()), 'clipped_text': 'ab',
        'clipped_complete': False, 'strict_rows': 0, 'strict_error': True,
        'source_ordinals': list(range(len(archive['tool_calls']))),
        'tool_ids_in_source_order': [t['id'] for t in archive['tool_calls']],
        'output_statuses': [t['output']['status'] for t in archive['tool_calls']],
        'success_values': [t['output']['success'] for t in archive['tool_calls']],
        'usage_origins': ['unknown', 'recorded_usage'],
        'manifest_duplicate_count': len(files) - 1 - len(set(files[n] for n in files if n.endswith('.json'))),
        'manifest_conflict_count': 1,
        'native_complete_requests': 2, 'native_incomplete_requests': 1,
        'native_whole_hash': hashlib.sha256(native).hexdigest(),
        'native_malformed_lines': 1, 'native_compactions': 1,
        'native_request_statuses': ['complete', 'complete', 'missing_blob'],
        'native_second_request_refs': ['a', 'b'],
        'native_logical_parent': 'attachment', 'native_first_assistant': 'new',
    }
    oracle = {'schema_version': 1, 'expected': expected,
              'fixture_hashes': {n: hashlib.sha256(b).hexdigest() for n, b in files.items()},
              'notes': ['Synthetic, no production content.',
                        'Manifest input is precisely the three archive files; ingest archive.json alone.',
                        'Clipping target is tool output result abé with maxCellBytes=3.',
                        'Native counts classify request status, not global scan diagnostics.']}
    (out / 'oracle.json').write_text(json.dumps(oracle, indent=2) + '\n')
    return oracle


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--out', type=Path, default=Path('/private/tmp/gmt015-persona/fixtures'))
    args = parser.parse_args()
    result = generate(args.out)
    print(json.dumps({'fixtures': str(args.out), 'files': list(result['fixture_hashes'])}))
