#!/usr/bin/env python3
"""Rerun the opening analyst turn with the same frozen corpus and optional extra guidance."""
import argparse
import json
from pathlib import Path
import shutil

from run import EXTENSION, HERE, MODELS, Timeline, dump, pi_turn, query_policy, sha, verify


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("original", type=Path)
    parser.add_argument("model", choices=MODELS)
    parser.add_argument("--batch", required=True, type=Path)
    parser.add_argument("--guidance", type=Path, help="Text appended to the saved opening prompt")
    parser.add_argument("--pi", default="pi")
    parser.add_argument("--extension", type=Path, default=EXTENSION)
    parser.add_argument("--timeout", type=int, default=180)
    args = parser.parse_args()
    original = args.original.resolve()
    args.batch = args.batch.resolve()
    meta = json.loads((original / "batch.json").read_text())
    prompt = (original / args.model / "analyst-1.prompt.md").read_text()
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
    meta.update(mode="opening-turn replay", compared_with=original.name,
                analyst_models=[args.model], prior_prompt_sha256=sha(original / args.model / "analyst-1.prompt.md"),
                guidance_supplied=bool(guidance))
    (args.batch / "batch.json").write_text(dump(meta) + "\n")
    work = args.batch / args.model
    (work / "artifacts").mkdir(parents=True)
    emit = Timeline(work, args.model)
    emit("host", "start", "Opening-turn replay", {"prior_batch": original.name, "guidance": guidance})
    policy = query_policy(args.batch, work, binary)
    try:
        pi_turn(args, work, emit, "analyst", prompt + ("\n\nAdditional guidance:\n" + guidance if guidance else ""),
                "analyst-1", args.model)
        check = verify(args.batch, work, binary, policy, "all", offline=False)
        emit("host", "replay", "Ledger matches the independent SQL" if check["passed"] else "Ledger did not pass replay", check)
        emit("host", "finish", "complete" if check["passed"] else "review needed")
        return int(not check["passed"])
    except (OSError, RuntimeError, ValueError) as error:
        emit("host", "failure", str(error))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
