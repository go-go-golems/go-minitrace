#!/usr/bin/env python3
"""Repeat the opening analyst prompt or recheck one stored ledger without a model call."""
import argparse
import json
from pathlib import Path
import shutil

from run import EXTENSION, HERE, MODELS, SCENARIO, Timeline, dump, pi_turn, query_policy, sha, verify


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("original", type=Path)
    parser.add_argument("model", choices=MODELS)
    parser.add_argument("--batch", required=True, type=Path)
    variants = parser.add_mutually_exclusive_group()
    variants.add_argument("--guidance", type=Path, help="Text appended to the saved opening prompt")
    variants.add_argument("--ledger-turn", type=int, choices=(1, 2, 3),
                          help="Recheck an existing turn's ledger snapshot without running Pi")
    parser.add_argument("--pi", default="pi")
    parser.add_argument("--extension", type=Path, default=EXTENSION)
    parser.add_argument("--timeout", type=int, default=180)
    args = parser.parse_args()
    original = args.original.resolve()
    args.batch = args.batch.resolve()
    meta = json.loads((original / "batch.json").read_text())
    prior = original / args.model
    source = (prior / f"ledger-turn-{args.ledger_turn}.js" if args.ledger_turn
              else prior / "analyst-1.prompt.md")
    saved = source.read_text()
    guidance = args.guidance.read_text() if args.guidance else ""
    args.thinking = meta["thinking"]
    for name, digest in meta["inputs"].items():
        if sha(original / name) != digest:
            raise RuntimeError(f"Pinned input changed: {name}")
    if meta["oracle_sha256"] != sha(HERE / "oracle.sql"):
        raise RuntimeError("The oracle changed since the original batch")
    args.batch.mkdir(mode=0o700, parents=True, exist_ok=False)
    shutil.copytree(original / "input", args.batch / "input")
    binary = args.batch / "input/go-minitrace"
    mode = "saved-ledger recheck" if args.ledger_turn else "opening-turn replay"
    meta.update(mode=mode, compared_with=original.name, analyst_models=[args.model],
                prior_source_sha256=sha(source), guidance_supplied=bool(guidance))
    (args.batch / "batch.json").write_text(dump(meta) + "\n")
    work = args.batch / args.model
    (work / "artifacts").mkdir(parents=True)
    emit = Timeline(work, args.model)
    emit("host", "start", mode, {"prior_batch": original.name, "guidance": guidance, "turn": args.ledger_turn})
    scope = SCENARIO["scope"] if args.ledger_turn == 3 else "all"
    try:
        if args.ledger_turn:
            shutil.copyfile(source, work / "artifacts/ledger.js")
        else:
            query_policy(args.batch, work, binary)
            prompt = saved + ("\n\nAdditional guidance:\n" + guidance if guidance else "")
            pi_turn(args, work, emit, "analyst", prompt, "analyst-1", args.model)
        check = verify(args.batch, work, binary, scope, offline=False)
        emit("host", "replay", "Ledger matches the independent SQL" if check["passed"] else "Ledger did not pass replay", check)
        emit("host", "finish", "complete" if check["passed"] else "review needed")
        return int(not check["passed"])
    except (OSError, RuntimeError, ValueError) as error:
        emit("host", "failure", str(error))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
