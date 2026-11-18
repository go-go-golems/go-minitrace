#!/usr/bin/env python3
"""Judge completed historical runs without rerunning analyst or stakeholder models."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import json
from pathlib import Path
import shutil
import subprocess

from judge import JUDGE_MODEL, JUDGE_THINKING, codex_judge, evidence_prompt, records, tool_evidence
from run import Timeline


def rejudge(args, source):
    model = source.name
    work = args.output / model
    work.mkdir()
    shutil.copy2(args.timeline / model / "events.jsonl", work / "events.jsonl")
    checks = [row["data"] for row in records(source / "events.jsonl") if row["kind"] == "replay"]
    if args.recheck:
        reference = args.recheck / model / "events.jsonl"
        if reference.is_file():
            checks += [dict(row["data"], turn=1, check_type="later isolated replay")
                       for row in records(reference) if row["kind"] == "replay"]
    tools = tool_evidence(source)
    (work / "api-evidence.json").write_text(json.dumps(tools, indent=2) + "\n")
    emit = Timeline(work, model)
    try:
        verdict = codex_judge(args, work, evidence_prompt(model, tools, checks), emit)
    except (OSError, RuntimeError, subprocess.TimeoutExpired) as error:
        emit("judge", "failure", str(error))
        return False
    emit("judge", "verdict", verdict, {"provider": "openai-codex", "model": args.judge_model,
                                     "thinking": args.judge_thinking, "review": "API and documentation"})
    return True


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("original", type=Path)
    parser.add_argument("--timeline", type=Path, help="Optional projected timeline with thinking markers")
    parser.add_argument("--recheck", type=Path, help="Optional saved opening ledger recheck")
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--codex", default="codex")
    parser.add_argument("--judge-model", default=JUDGE_MODEL)
    parser.add_argument("--judge-thinking", choices=("low", "medium", "high"), default=JUDGE_THINKING)
    parser.add_argument("--timeout", type=int, default=180)
    args = parser.parse_args()
    args.original = args.original.resolve()
    args.timeline = args.timeline.resolve() if args.timeline else args.original
    args.output = args.output.resolve()
    args.output.mkdir(mode=0o700, parents=True, exist_ok=False)
    original_meta = json.loads((args.original / "batch.json").read_text())
    original_meta.update(mode="API and documentation rejudge", original_batch=args.original.name,
                         judge={"provider": "openai-codex", "model": args.judge_model,
                                "thinking": args.judge_thinking})
    (args.output / "batch.json").write_text(json.dumps(original_meta, indent=2) + "\n")
    sources = [path.parent for path in sorted(args.original.glob("*/events.jsonl"))]
    if not sources:
        parser.error("No runs found")
    with ThreadPoolExecutor(max_workers=2) as pool:
        return int(not all(list(pool.map(lambda source: rejudge(args, source), sources))))


if __name__ == "__main__":
    raise SystemExit(main())
