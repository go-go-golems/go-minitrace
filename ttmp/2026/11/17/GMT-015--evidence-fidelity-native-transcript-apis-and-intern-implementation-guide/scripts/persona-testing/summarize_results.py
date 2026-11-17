#!/usr/bin/env python3
"""Independently grade finished persona runs; never execute model-authored commands.

A receipt is final only when elapsed_seconds exists. D claims remain provisional
until separately reviewed command replay receipts are provided by the operator.
"""
import argparse
from collections import defaultdict
from datetime import datetime, timezone
import hashlib
import importlib.util
import json
from pathlib import Path
import sys

sys.dont_write_bytecode = True
spec = importlib.util.spec_from_file_location('persona_grade', Path(__file__).with_name('grade.py'))
grader = importlib.util.module_from_spec(spec)
spec.loader.exec_module(grader)


def load(path):
    return json.loads(path.read_text())


def check(key, actual, expected):
    return {'key': key, 'passed': grader.same(actual, expected), 'actual': actual, 'expected': expected}


def grade_a(answer, corpus):
    checks = []
    rows = answer.get('sessions', [])
    if not isinstance(rows, list):
        rows = []
    actual = {r.get('session_id'): r for r in rows if isinstance(r, dict)}
    checks.append(check('session_ids', sorted(str(k) for k in actual), sorted(r['session_id'] for r in corpus)))
    checks.append(check('session_rows', len(rows), len(corpus)))
    for row in corpus:
        got = actual.get(row['session_id'], {})
        for field in ['turn_count', 'tool_count']:
            checks.append(check(row['session_id'] + '.' + field, got.get(field), row[field]))
    return checks


def grade_c(answer, corpus):
    checks = [check('exact_replay_possible', answer.get('exact_replay_possible'), False)]
    scans = answer.get('real_scans', [])
    if not isinstance(scans, list):
        scans = []
    actual = {str(Path(r.get('path', '')).resolve()): r for r in scans if isinstance(r, dict) and isinstance(r.get('path'), str)}
    checks.append(check('real_scan_paths', sorted(actual), sorted(r['native_path'] for r in corpus)))
    checks.append(check('real_scan_rows', len(scans), len(corpus)))
    for row in corpus:
        got = actual.get(row['native_path'], {})
        raw = Path(row['native_path']).read_bytes()
        digest = hashlib.sha256(raw).hexdigest()
        if digest != row['native_sha256'] or len(raw) != row['native_bytes']:
            raise ValueError('corpus native drift: ' + row['native_path'])
        expected = {'sha256': digest, 'bytes': len(raw), 'lines': len(raw.splitlines()), 'hashScope': 'whole_source'}
        for key, value in expected.items():
            checks.append(check(Path(row['native_path']).name + '.' + key, got.get(key), value))
    return checks


def summarize(matrix, oracle, manifest):
    design = load(matrix / 'design.json')
    corpus = load(manifest)['rows']
    # Corpus metadata is not blindly trusted: verify exact archives and actual arrays.
    for row in corpus:
        raw = Path(row['archive_path']).read_bytes()
        if hashlib.sha256(raw).hexdigest() != row['archive_sha256']:
            raise ValueError('corpus archive drift: ' + row['archive_path'])
        archive = json.loads(raw)
        row['turn_count'] = len(archive.get('turns') or [])
        row['tool_count'] = len(archive.get('tool_calls') or [])
    runs = []
    for receipt_path in sorted(matrix.glob('*/receipt.json')):
        try:
            receipt = load(receipt_path)
        except (OSError, ValueError):
            continue  # receipt can briefly be truncated while runner finishes writing
        run = {k: receipt.get(k) for k in ['run_id', 'task', 'model', 'requested_thinking', 'repeat',
                'elapsed_seconds', 'exit_code', 'tool_calls', 'tool_error_events', 'reported_usage', 'observed_models']}
        run['finished'] = 'elapsed_seconds' in receipt
        run['timed_out'] = receipt.get('timed_out', False)
        if not run['finished']:
            runs.append(run)
            continue
        directory = receipt_path.parent
        run['artifacts'] = {name: (directory / name).is_file() and (directory / name).stat().st_size > 0 for name in ['answer.json', 'report.md']}
        run['source_artifact_count'] = sum(1 for p in directory.rglob('*') if p.is_file() and p.suffix in ['.js', '.sql'] and 'sessions' not in p.parts)
        checks = []
        try:
            answer = load(directory / 'answer.json')
            if not isinstance(answer, dict):
                raise ValueError('answer must be object')
            if run['task'] == 'A':
                checks = grade_a(answer, corpus)
                run['unverified_claims'] = {k: answer.get(k) for k in ['sql_js_equal', 'frameworks', 'conversion_validation']}
            elif run['task'] == 'B':
                checks = grader.grade(directory / 'answer.json', oracle, ['fidelity', 'provenance'])['checks']
            elif run['task'] == 'C':
                checks = grader.grade(directory / 'answer.json', oracle, ['native'])['checks'] + grade_c(answer, corpus)
            elif run['task'] == 'D':
                keys = ['good_command_succeeded', 'bad_sibling_diagnosed', 'required_source_visible', 'runtime_flags_visible', 'negative_default_supported']
                run['unverified_claims'] = {k: answer.get(k) for k in keys + ['selected_session_ids']}
                run['claim_schema_valid'] = all(type(answer.get(k)) is bool for k in keys)
                run['replay_status'] = 'pending_safe_manual_review'
                replay = directory / 'independent-replay.json'
                if replay.exists():
                    run['replay_receipt'] = load(replay)
                    run['replay_status'] = 'separate_receipt_available_not_automatically_scored'
            run['answer_parseable'] = True
        except (OSError, ValueError, KeyError, TypeError) as exc:
            run['answer_parseable'] = False
            run['grading_error'] = str(exc)
        run['checks'] = checks
        run['correct_fields'] = sum(c['passed'] for c in checks)
        run['graded_fields'] = len(checks)
        run['all_graded_fields_correct'] = bool(checks) and all(c['passed'] for c in checks)
        runs.append(run)
    buckets = defaultdict(lambda: {'runs_started': 0, 'runs_finished': 0, 'answers_present': 0, 'reports_present': 0, 'timeouts': 0, 'correct_fields': 0, 'graded_fields': 0, 'tool_calls': 0, 'tool_error_events': 0, 'elapsed_seconds': 0, 'reported_usage': {}})
    for run in runs:
        key = '/'.join(str(run[k]) for k in ['model', 'requested_thinking', 'task'])
        b = buckets[key]; b['runs_started'] += 1
        if not run['finished']:
            continue
        b['runs_finished'] += 1
        for out, name in [('answers_present','answer.json'), ('reports_present','report.md')]:
            b[out] += bool(run['artifacts'][name])
        b['timeouts'] += run['timed_out']
        for key in ['correct_fields', 'graded_fields', 'tool_calls', 'tool_error_events', 'elapsed_seconds']:
            b[key] += run.get(key) or 0
        for key, value in (run.get('reported_usage') or {}).items():
            b['reported_usage'][key] = b['reported_usage'].get(key, 0) + value
    finished = sum(r['finished'] for r in runs)
    return {'schema': 'gmt015-persona-independent-grades/v1', 'generated_at': datetime.now(timezone.utc).isoformat(),
            'status': 'complete' if finished == len(design['specs']) else 'partial',
            'planned_runs': len(design['specs']), 'started_runs': len(runs), 'finished_runs': finished,
            'matrix_revision': design['revision'], 'aggregates': dict(buckets), 'runs': runs,
            'limits': ['Field correctness is not task completion or proof of tool use; inspect execution traces.',
                       'D booleans are unverified model claims, excluded from correct-field scores.',
                       'A SQL/JS equality and conversion validation claims require trace review.',
                       'Requested thinking labels are not guaranteed backend reasoning settings.',
                       'Purposeful historical corpus selection is not representative sampling.',
                       'No model-authored code is executed by this summarizer.']}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--matrix', type=Path, default=Path('/private/tmp/gmt015-persona/matrix'))
    p.add_argument('--oracle', type=Path, default=Path('/private/tmp/gmt015-persona/fixtures/oracle.json'))
    p.add_argument('--manifest', type=Path, default=Path('/private/tmp/gmt015-persona/corpus/manifest.json'))
    p.add_argument('--output', type=Path, required=True)
    a = p.parse_args()
    result = summarize(a.matrix, a.oracle, a.manifest)
    a.output.parent.mkdir(parents=True, exist_ok=True)
    a.output.write_text(json.dumps(result, indent=2) + '\n')
    print(json.dumps({k: result[k] for k in ['status','planned_runs','started_runs','finished_runs']}))


if __name__ == '__main__':
    main()
