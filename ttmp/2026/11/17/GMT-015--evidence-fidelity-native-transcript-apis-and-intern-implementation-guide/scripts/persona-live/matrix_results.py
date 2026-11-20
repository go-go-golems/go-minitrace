"""Summarize sanitized receipts from a complete set of matrix batch directories."""
import argparse
import json
from pathlib import Path
import re

from compare import batch_summary
from judge import records
from matrix import LEVELS, TASKS
from run import MODELS


def summarize(paths, retries=()):
    inputs = judge = None
    cells = {}
    tasks = {}
    interrupted = []
    for path in [*paths, *retries]:
        meta = json.loads((path / "batch.json").read_text())
        task, level = meta["scenario"], meta["thinking"]
        batch = batch_summary(path)
        if inputs is None:
            inputs, judge = batch["inputs"], batch["judge"]
        if inputs != batch["inputs"] or judge != batch["judge"]:
            raise ValueError(f"Binary, archive or judge mismatch in {path}")
        contract = {key: meta[key] for key in ("scenario_sha256", "oracle_sha256")}
        if tasks.setdefault(task, contract) != contract:
            raise ValueError(f"Scenario mismatch for {task}")
        for model, run in batch["runs"].items():
            session = path / model / "analyst-session.jsonl"
            model_settings, effort_settings = set(), set()
            if session.is_file():
                for event in records(session):
                    if event.get("type") == "model_change":
                        model_settings.add((event.get("provider"), event.get("modelId")))
                    elif event.get("type") == "thinking_level_change":
                        effort_settings.add(event.get("thinkingLevel"))
            settings_match = model_settings == {("lunaroute", model)} and effort_settings == {level}
            if run["terminal_event"] == "finish" and not settings_match:
                raise ValueError(f"Recorded analyst settings mismatch in {path}/{model}")
            key = (task, model, level)
            if path in retries:
                original = cells.get(key)
                if not original or original["terminal_event"] != "failure":
                    raise ValueError(f"Retry requires a recorded failed attempt for {key}")
                interrupted.append(original)
            elif key in cells:
                raise ValueError(f"Duplicate run {key}")
            opening = path / model / "ledger-turn-1.js"
            source = opening.read_text() if opening.is_file() else ""
            cells[key] = dict(opening_ledger_has_tagged_sql=bool(re.search(r"\.sql\s*`", source)),
                              opening_ledger_has_fluent_from=bool(re.search(r"\.from\s*\(", source)),
                              task=task, analyst=model, effort=level, batch=path.name,
                              recorded_pi_settings_match=settings_match,
                              **{k: v for k, v in run.items() if k != "by_phase"})
    expected = {(t, m, e) for t in TASKS for m in MODELS for e in LEVELS}
    if cells.keys() != expected:
        raise ValueError(f"Matrix cell mismatch: {cells.keys() ^ expected}")
    for key, run in cells.items():
        if run["terminal_event"] not in ("finish", "failure"):
            raise ValueError(f"Unfinished run {key}")
    return dict(inputs=inputs, judge=judge, tasks=tasks, prior_failed_attempts=interrupted, runs=[cells[k] for k in sorted(cells)])


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("batches", type=Path, nargs="+")
    parser.add_argument("--retry", type=Path, action="append", default=[], help="Fresh retry batch for recorded failures")
    args = parser.parse_args()
    print(json.dumps(summarize(args.batches, args.retry), indent=2))
