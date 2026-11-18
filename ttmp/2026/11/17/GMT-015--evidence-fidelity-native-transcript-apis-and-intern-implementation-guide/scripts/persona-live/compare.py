#!/usr/bin/env python3
"""Compare observed persona results across two frozen batches."""
import argparse
import json
from pathlib import Path

from judge import records, tool_evidence


def milestones(work):
    calls = {}
    ordinal = 0
    first_js = first_ledger = None
    for phase in sorted(work.glob("analyst-*.pi.jsonl")):
        for event in records(phase):
            if event["type"] == "tool_execution_start":
                ordinal += 1
                if event.get("toolName") == "minitrace":
                    calls[event["toolCallId"]] = (ordinal, event.get("args") or {})
            elif event["type"] == "tool_execution_end" and event.get("toolName") == "minitrace":
                call = calls.get(event["toolCallId"])
                if not call or call[1].get("kind") != "js":
                    continue
                try:
                    result = json.loads(event["result"]["content"][0]["text"])
                except (ValueError, IndexError, KeyError, TypeError):
                    continue
                if not isinstance(result, dict) or result.get("exit") != 0:
                    continue
                first_js = call[0] if first_js is None else min(first_js, call[0])
                if call[1].get("path") == ["ledger"] or call[1].get("text") == "ledger":
                    first_ledger = call[0] if first_ledger is None else min(first_ledger, call[0])
    return {"first_successful_js_call_ordinal": first_js, "first_successful_ledger_call_ordinal": first_ledger}


def batch_summary(batch):
    meta = json.loads((batch / "batch.json").read_text())
    runs = {}
    for path in sorted(batch.glob("*/events.jsonl")):
        events = list(records(path))
        if not events:
            continue
        work = path.parent
        replays = [
            {key: event["data"].get(key) for key in
             ("turn", "scope", "passed", "observed_rows", "expected_rows", "missing_count", "unexpected_count")}
            for event in events if event["kind"] == "replay"
        ]
        runs[work.name] = {
            **milestones(work), **tool_evidence(work),
            "elapsed_seconds": round(events[-1]["time"] - events[0]["time"], 2),
            "terminal_event": events[-1]["kind"], "replays": replays,
            "judge_delivered": any(event["kind"] == "verdict" for event in events),
        }
    return {"batch": batch.name, "inputs": meta["inputs"], "runs": runs, "judge": meta.get("judge")}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("baseline", type=Path)
    parser.add_argument("candidate", type=Path)
    args = parser.parse_args()
    print(json.dumps({"baseline": batch_summary(args.baseline), "candidate": batch_summary(args.candidate)}, indent=2))
