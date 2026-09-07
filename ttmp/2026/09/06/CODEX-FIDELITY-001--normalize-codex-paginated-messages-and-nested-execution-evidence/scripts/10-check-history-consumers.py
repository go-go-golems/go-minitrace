#!/usr/bin/env python3
"""End-to-end checks against a converted paginated-fidelity synthetic archive.
Usage: python3 10-check-history-consumers.py '/tmp/archive/active/*/*.minitrace.json'
Runs only the checkout CLI, never historical transcript code.
"""
import json
import subprocess
import sys


def query(*args):
    result = subprocess.run(['go', 'run', './cmd/go-minitrace', 'query', *args,
                             '--archive-glob', sys.argv[1], '--output', 'json'],
                            check=True, capture_output=True, text=True)
    return json.loads(result.stdout)


first = query('commands', 'history', 'file-history', '--path', 'first.txt')[0]
assert len(first['timeline']) == 1
row = first['timeline'][0]
assert row['success'] is None and row['evidence_status'] == 'attempted'
assert row['preceding_instruction'] is None
assert row['match_source'] == 'structural_file_target'
assert first['summary'][0]['created_before_visible_history'] is None
assert first['summary'][0]['confirmed_effects'] == 0
second = query('commands', 'history', 'file-history', '--path', 'second.txt')[0]
assert len(second['timeline']) == 1
assert second['timeline'][0]['success'] is None
assert query('commands', 'history', 'file-history', '--path', 'never-created')[0]['timeline'] == []
counts = query('run', '--sql', 'SELECT tool_call_count, tool_call_record_count, orchestration_count, execution_record_count, file_change_count, model_invocation_count, file_touch_count, confirmed_file_target_count FROM sessions')[0]
assert sum(counts[k] for k in ('tool_call_record_count', 'orchestration_count', 'execution_record_count', 'file_change_count')) == counts['tool_call_count']
assert counts['execution_record_count'] == 6 and counts['file_touch_count'] == 2
assert counts['confirmed_file_target_count'] == 0
print(json.dumps(dict(passed=True, first_target_rows=1, second_target_rows=1, false_branch_rows=0, counts=counts), indent=2))
