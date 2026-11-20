"""Check a grouped aggregate under the saved Seatbelt policy and normal local CLI."""
import argparse
import json
from pathlib import Path
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("batch", type=Path)
    parser.add_argument("model")
    args = parser.parse_args()
    batch = args.batch.resolve()
    work = batch / args.model
    for predicate in ("success IS NULL", "result IS NULL", "COALESCE(result,'')=''"):
        query = f"SELECT session_id, tool_name, SUM({predicate}) AS n FROM tool_calls GROUP BY session_id, tool_name"
        for profiled in (True, False):
            argv = ["/usr/bin/sandbox-exec", "-f", str(work / "query.sb")] if profiled else []
            argv += [str(batch / "input/go-minitrace"), "query", "run", "--sql", query,
                     "--archive-glob", str(batch / "input/archives/*.minitrace.json"),
                     "--cache-mode", "memory", "--result-envelope", "--output", "json"]
            proc = subprocess.run(argv, cwd=work, capture_output=True, text=True, timeout=30)
            result = {"predicate": predicate, "seatbelt": profiled, "exit": proc.returncode}
            if proc.returncode:
                result["error"] = proc.stderr.strip()
            else:
                envelope, = json.loads(proc.stdout)
                result.update(count=envelope["count"], complete=envelope["complete"])
            print(json.dumps(result))


if __name__ == "__main__":
    main()
