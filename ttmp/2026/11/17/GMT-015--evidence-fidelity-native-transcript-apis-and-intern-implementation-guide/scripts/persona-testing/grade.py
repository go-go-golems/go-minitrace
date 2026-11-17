#!/usr/bin/env python3
"""Grade actual answer.json values against a private synthetic-fixture oracle.

Exit 0 passes, 1 is a wrong/missing answer, 2 is invalid input/fixture drift.
No LLM self-evaluation or prose interpretation is used. Extra answer keys are ignored.
This verifies results, not whether the agent used an allowed tool; audit traces separately.
"""
import argparse
import hashlib
import json
from pathlib import Path
import sys

GROUPS = {
    'fidelity': ['unicode_bytes', 'clipped_text', 'clipped_complete', 'strict_rows', 'strict_error'],
    'provenance': ['source_ordinals', 'manifest_duplicate_count', 'manifest_conflict_count'],
    'native': ['native_complete_requests', 'native_incomplete_requests', 'native_whole_hash'],
    'semantics': ['tool_ids_in_source_order', 'output_statuses', 'success_values', 'usage_origins'],
    'native-detail': ['native_malformed_lines', 'native_compactions', 'native_request_statuses',
                      'native_second_request_refs', 'native_logical_parent', 'native_first_assistant'],
}


def same(actual, expected):
    # Python True == 1; explicitly disallow that false positive, recursively.
    if type(actual) is not type(expected):
        return False
    if isinstance(expected, list):
        return len(actual) == len(expected) and all(same(a, e) for a, e in zip(actual, expected))
    if isinstance(expected, dict):
        return actual.keys() == expected.keys() and all(same(actual[k], v) for k, v in expected.items())
    return actual == expected


def grade(answer_path, oracle_path, groups):
    oracle = json.loads(oracle_path.read_text())
    if oracle.get('schema_version') != 1:
        raise ValueError('unsupported oracle schema')
    for name, expected_hash in oracle['fixture_hashes'].items():
        if Path(name).name != name:
            raise ValueError('fixture name must be a basename')
        digest = hashlib.sha256((oracle_path.parent / name).read_bytes()).hexdigest()
        if digest != expected_hash:
            raise ValueError(f'fixture drift: {name}')
    answer = json.loads(answer_path.read_text())
    if not isinstance(answer, dict):
        raise ValueError('answer must be a JSON object')
    keys = list(dict.fromkeys(k for group in groups for k in GROUPS[group]))
    checks = [{'key': k, 'present': k in answer,
               'passed': k in answer and same(answer[k], oracle['expected'][k]),
               'expected': oracle['expected'][k], 'actual': answer.get(k)} for k in keys]
    return {'schema_version': 1, 'answer': str(answer_path), 'groups': groups,
            'passed': all(c['passed'] for c in checks),
            'passed_checks': sum(c['passed'] for c in checks), 'total_checks': len(checks),
            'checks': checks, 'fixture_integrity': 'verified'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--answer', type=Path, required=True)
    parser.add_argument('--oracle', type=Path, default=Path('/private/tmp/gmt015-persona/fixtures/oracle.json'))
    parser.add_argument('--group', action='append', choices=[*GROUPS, 'all'], default=None)
    parser.add_argument('--output', type=Path)
    args = parser.parse_args()
    groups = args.group or ['all']
    if 'all' in groups:
        groups = list(GROUPS)
    try:
        result = grade(args.answer, args.oracle, groups)
        code = 0 if result['passed'] else 1
    except (OSError, ValueError, KeyError, TypeError) as exc:
        result = {'passed': False, 'grader_error': str(exc)}
        code = 2
    text = json.dumps(result, indent=2) + '\n'
    if args.output:
        args.output.write_text(text)
    print(text, end='')
    return code


if __name__ == '__main__':
    sys.exit(main())
