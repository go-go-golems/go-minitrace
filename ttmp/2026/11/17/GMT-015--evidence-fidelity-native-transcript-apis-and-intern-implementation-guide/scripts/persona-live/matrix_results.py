"""Summarize sanitized receipts from a complete set of matrix batch directories."""
import argparse
import json
from pathlib import Path

from compare import batch_summary
from matrix import LEVELS, TASKS
from run import MODELS


def summarize(paths):
    inputs = judge = None
    cells = {}
    tasks = {}
    for path in paths:
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
            key = (task, model, level)
            if key in cells:
                raise ValueError(f"Duplicate run {key}")
            cells[key] = dict(task=task, analyst=model, effort=level, batch=path.name,
                              **{k: v for k, v in run.items() if k != "by_phase"})
    expected = {(t, m, e) for t in TASKS for m in MODELS for e in LEVELS}
    if cells.keys() != expected:
        raise ValueError(f"Matrix cell mismatch: {cells.keys() ^ expected}")
    for key, run in cells.items():
        if run["terminal_event"] not in ("finish", "failure"):
            raise ValueError(f"Unfinished run {key}")
    return dict(inputs=inputs, judge=judge, tasks=tasks, runs=[cells[k] for k in sorted(cells)])


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("batches", type=Path, nargs="+")
    args = parser.parse_args()
    print(json.dumps(summarize(args.batches), indent=2))
