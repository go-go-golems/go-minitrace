"""Verify that changing the submitted helper exchanges the two observed counts."""
import argparse
from collections import Counter
import json
from pathlib import Path
import re
import tempfile

from run import command_json, input_glob, query_policy, result_rows, sha

FIELDS = ("session_id", "calls", "successful", "failed")
# This bounded edit targets the named, two-argument declaration required by the brief.
HELPER = re.compile(r"^\s*function\s+withSuccess\s*\(\s*\w+\s*,\s*(\w+)\s*\)\s*\{", re.MULTILINE)


def check_turn(batch, work, binary, turn, scope):
    original = work / f"ledger-turn-{turn}.js"
    text = original.read_text()
    matches = list(HELPER.finditer(text))
    if len(matches) != 1:
        raise ValueError(f"Expected one two-argument withSuccess declaration in {original}")
    match, = matches
    value = match.group(1)
    # Preserve unknown/null and any other arguments. Only swap the numeric known outcomes.
    change = f"\nif ({value} === 1) {value} = 0; else if ({value} === 0) {value} = 1;\n"
    with tempfile.TemporaryDirectory(dir=work) as temp:
        candidate = Path(temp)
        (candidate / "ledger.js").write_text(text[:match.end()] + change + text[match.end():])
        policy = query_policy(batch, work, binary, candidate)
        raw = command_json(["/usr/bin/sandbox-exec", "-f", str(policy), str(binary), "query", "commands",
                            "--query-repository", str(candidate), "ledger", "--archive-glob",
                            input_glob(batch, scope), "--output", "json"], work)
    rows = result_rows(raw, FIELDS)
    expected = json.loads((work / f"expected-{turn}-{scope}.json").read_text())["rows"]
    actual = Counter(tuple(row[f] for f in FIELDS) for row in rows)
    control = Counter(tuple(row[f] for f in FIELDS) for row in expected)
    swapped = Counter((r["session_id"], r["calls"], r["failed"], r["successful"]) for r in expected)
    return dict(turn=turn, scope=scope, source_sha256=sha(original), rows=len(rows),
                discriminating_sessions=sum(r["successful"] != r["failed"] for r in expected),
                passed=actual == swapped and swapped != control)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("batches", nargs="+", type=Path)
    args = parser.parse_args()
    reports = []
    for batch in args.batches:
        batch = batch.resolve()
        meta = json.loads((batch / "batch.json").read_text())
        if meta["scenario"] != "composition":
            raise ValueError("Expected a composition batch")
        binary = batch / "input/go-minitrace"
        for model in meta["analyst_models"]:
            work = batch / model
            checks = [check_turn(batch, work, binary, turn, scope)
                      for turn, scope in enumerate(("all", "all", "claude-code"), 1)]
            reports.append(dict(analyst=model, thinking=meta["thinking"], checks=checks))
    print(json.dumps(reports, indent=2))
    return int(any(not check["passed"] for run in reports for check in run["checks"]))


if __name__ == "__main__":
    raise SystemExit(main())
