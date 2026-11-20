"""Run the four-task, two-effort persona matrix using run.py's two analysts."""
import argparse
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
import subprocess
import sys

TASKS = ("file-change-ledger", "recovery", "efficiency", "coverage")
LEVELS = ("low", "high")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", required=True, type=Path, help="Fresh parent for generated batches")
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--corpus", required=True, type=Path)
    parser.add_argument("--omit", nargs="*", choices=[f"{t}-{e}" for t in TASKS for e in LEVELS], default=[])
    parser.add_argument("--jobs", type=int, choices=(1, 2, 3), default=2)
    parser.add_argument("--timeout", type=int, default=360)
    args = parser.parse_args()
    args.root.mkdir(mode=0o700, parents=True, exist_ok=False)
    def run(cell):
        task, level = cell
        return subprocess.run([sys.executable, str(Path(__file__).with_name("run.py")),
            "--batch", str(args.root / f"{task}-{level}"), "--binary", str(args.binary),
            "--corpus", str(args.corpus), "--scenario", task, "--thinking", level,
            "--timeout", str(args.timeout)], check=False).returncode

    cells = [(task, level) for task in TASKS for level in LEVELS if f"{task}-{level}" not in args.omit]
    with ThreadPoolExecutor(max_workers=args.jobs) as pool:
        statuses = list(pool.map(run, cells))
    for cell, status in zip(cells, statuses):
        print(*cell, status, flush=True)
    return int(any(statuses))


if __name__ == "__main__":
    raise SystemExit(main())
