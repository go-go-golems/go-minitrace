#!/usr/bin/env python3
"""Unscored host probe: replay completed D commands on a zero-tool archive.

Uses fixed pinned-binary argv and the original per-run sandbox policy. Never
modifies model output, original host_replay, or grades. Existing receipts are
reused so this can be rerun after more cohort members finish.
"""
import argparse
import json
from pathlib import Path
from common import load, save, verify
from record_command import execute

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--bundle', type=Path, required=True)
    parser.add_argument('--cohort', required=True)
    args = parser.parse_args()
    bundle = args.bundle.resolve()
    if not args.cohort or Path(args.cohort).name != args.cohort:
        parser.error('cohort must be one path component')
    verify(bundle)
    session_id = '01a1364a-f58a-7ce3-8460-a856dacc8a1a'
    source = bundle / 'corpus/archives/codex/active/2026-10' / (session_id + '.minitrace.json')
    if not source.is_file():
        parser.error('zero-tool source is missing')
    results = []
    for receipt_path in sorted((bundle / 'runs' / args.cohort).glob('D-*/receipt.json')):
        original = load(receipt_path)
        if 'elapsed_seconds' not in original:
            continue
        run = receipt_path.parent
        policy = bundle / 'control' / (args.cohort + '-' + run.name + '.sb')
        output = bundle / 'control/zero-tool-probes' / args.cohort / run.name / 'command.json'
        if output.exists():
            receipt = load(output)
        else:
            argv = ['/usr/bin/sandbox-exec', '-f', str(policy), str(bundle / 'bin/go-minitrace'),
                    'query', 'commands', '--query-repository', str(run / 'commands'),
                    'good', '--source', str(source), '--output', 'json']
            receipt = execute(argv, run, output, 60)
        value = None
        error = None
        try:
            value = load(receipt['stdout_path'])
        except (OSError, ValueError) as exc:
            error = str(exc)
        passed = (receipt['exit_code'] == 0 and not receipt['timed_out']
                  and isinstance(value, list) and len(value) == 1
                  and isinstance(value[0], dict)
                  and value[0].get('session_id') == session_id
                  and type(value[0].get('tool_count')) is int and value[0]['tool_count'] == 0)
        results.append({'run_id': run.name, 'unscored': True, 'passed': passed,
                        'final_answer_present': (run / 'answer.json').is_file(),
                        'code_state': 'final-answer-present' if (run / 'answer.json').is_file() else 'unfinished-generated-code',
                        'exit_code': receipt['exit_code'], 'value': value,
                        'parse_error': error, 'receipt_path': str(output)})
    save(bundle / 'control/zero-tool-probes' / args.cohort / 'summary.json', results)
    print(json.dumps(results, indent=2))

if __name__ == '__main__':
    main()
